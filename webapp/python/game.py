import asyncio
from collections import defaultdict, namedtuple
import logging
import os
import sys
import time

# namedtuple を dict として出力するために標準ライブラリの json ではなく
# simplejson を使います。
import simplejson
import MySQLdb


# types for JSON
Schedule = namedtuple("Schedule", ("time", "milli_isu", "total_power"))
Item = namedtuple("Item", ("item_id", "count_bought", "count_built", "next_price", "power", "building"))
OnSale = namedtuple("OnSale", ("item_id", "time"))
Building = namedtuple("Building", ("time", "count_built", "power"))
GameStatus = namedtuple("GameStatus", ("time", "adding", "schedule", "items", "on_sale"))
Adding = namedtuple("Adding", ("time", "isu"))
Buying = namedtuple("Buying", ("item_id", "ordinal", "time"))


_db_info = None

def connect_db():
    """MySQLに接続して connection object を返す"""
    global _db_info
    if _db_info is None:
        host = os.environ.get("ISU_DB_HOST", "127.0.0.1")
        port = int(os.environ.get("ISU_DB_PORT", "3306"))
        user = os.environ.get("ISU_DB_USER", "root")
        passwd = os.environ.get("ISU_DB_PASSWORD", "")
        _db_info = {
            "host": host,
            "port": port,
            "user": user,
            "password": passwd,
            "charset": "utf8mb4",
            "db": "isudb",
        }
    return MySQLdb.connect(**_db_info)


def initialize():
    conn = connect_db()
    try:
        cur = conn.cursor()
        cur.execute("TRUNCATE TABLE adding")
        cur.execute("TRUNCATE TABLE buying")
        cur.execute("TRUNCATE TABLE room_time")
    finally:
        conn.close()


def calc_item_power(m: dict, count : int) -> int:
    """アイテムマスタ m から count 個目のそのアイテムの生産力を計算する"""
    a = m['power1']
    b = m['power2']
    c = m['power3']
    d = m['power4']
    return (c * count + 1) * (d ** (a * count + b))


def calc_item_price(m: dict, count : int) -> int:
    """アイテムマスタ m から count 個目のそのアイテムの価格を計算する"""
    a = m['price1']
    b = m['price2']
    c = m['price3']
    d = m['price4']
    return (c * count + 1) * (d ** (a * count + b))


# JSON中で利用する10進指数表記
# [x, y] = x * 10^y
def int2exp(x: int) -> (int, int):
    s = str(x)
    if not s:
        return (0, 0)
    if len(s) <= 15:
        return (x, 0)
    return (int(s[:15]), len(s)-15)


def calc_status(current_time: int, mitems: dict, addings: list, buyings: list):
    # 1ミリ秒に生産できる椅子の単位をミリ椅子とする
    total_milli_isu : int = 0
    total_power : int = 0

    item_power = {itemID: 0 for itemID in mitems}  # ItemID: power
    item_price = {}  # ItemID: price
    item_on_sale = {}  # ItemID: on_sale
    item_built = defaultdict(int)  # ItemID: BuiltCount
    item_bought = defaultdict(int)
    item_building = {itemID: [] for itemID in mitems}

    item_power0 = {}
    item_built0 = {}

    adding_at = {}
    buying_at = defaultdict(list)

    for a in addings:
        if a.time <= current_time:
            total_milli_isu += int(a.isu) * 1000
        else:
            adding_at[a.time] = a

    for b in buyings:
        m = mitems[b.item_id]
        item_bought[b.item_id] += 1
        total_milli_isu -= calc_item_price(m, b.ordinal) * 1000

        if b.time <= current_time:
            item_built[b.item_id] += 1
            power = calc_item_power(m, item_bought[b.item_id])
            item_power[b.item_id] += power
            total_power += power
            total_milli_isu += power * (current_time - b.time)
        else:
            buying_at[b.time].append(b)

    for item_id, m in mitems.items():
        item_power0[item_id] = int2exp(item_power[item_id])
        item_built0[item_id] = item_built[item_id]
        price = calc_item_price(m, item_bought[item_id]+1)
        item_price[item_id] = price
        if total_milli_isu >= price*1000:
            # 0 は 時刻 currentTime で購入可能であることを表す
            item_on_sale[item_id] = 0

    # current_time の状態
    schedule = [Schedule(current_time, int2exp(total_milli_isu), int2exp(total_power))]

    # current_time+1000 までの状態
    for t in range(current_time+1, current_time+1001):
        total_milli_isu += total_power
        updated = False

        if t in adding_at:
            updated = True
            total_milli_isu += int(adding_at[t].isu) * 1000

        if t in buying_at:
            updated = True
            updated_ids = set()

            for b in buying_at[t]:
                m = mitems[b.item_id]
                updated_ids.add(b.item_id)
                item_built[b.item_id] += 1

                power = calc_item_power(m, b.ordinal)
                item_power[b.item_id] += power
                total_power += power

            for id in updated_ids:
                item_building[id].append(
                    Building(t, item_built[id], int2exp(item_power[id]))
                )

        if updated:
            schedule.append(
                Schedule(t, int2exp(total_milli_isu), int2exp(total_power)),
            )

        # 時刻 t で購入可能になったアイテムを記録する
        for id in mitems:
            if id in item_on_sale:
                continue
            if total_milli_isu >= item_price[id] * 1000:
                item_on_sale[id] = t

    gs_addings = list(adding_at.values())

    gs_items = [
        Item(
            item_id,
            item_bought[item_id],
            item_built0[item_id],
            int2exp(item_price[item_id]),
            item_power0[item_id],
            item_building[item_id],
        ) for item_id in mitems]

    gs_on_sale = [OnSale(id, t) for id, t in item_on_sale.items()]

    return GameStatus(
        0,
        gs_addings,
        schedule,
        gs_items,
        gs_on_sale)


def update_room_time(conn, room_name: str, req_time: int) -> int:
    """部屋のロックを取りタイムスタンプを更新する

    トランザクション開始後この関数を呼ぶ前にクエリを投げると、
    そのトランザクション中の通常のSELECTクエリが返す結果がロック取得前の
    状態になることに注意 (keyword: MVCC, repeatable read).
    """
    cur = conn.cursor()

    # See page 13 and 17 in https://www.slideshare.net/ichirin2501/insert-51938787
    cur.execute("INSERT INTO room_time(room_name, time) VALUES (%s, 0) ON DUPLICATE KEY UPDATE time = time",
                (room_name,))

    cur.execute("SELECT time FROM room_time WHERE room_name = %s FOR UPDATE", (room_name,))
    room_time = cur.fetchone()[0]

    current_time = get_current_time(conn)

    if room_time > current_time:
        raise RuntimeError(f"room_time is future: room_time={room_time}, req_time={req_time}")

    if req_time and req_time < current_time:
        raise RuntimeError(f"req_time is past: req_time={req_time}, current_time={current_time}")

    cur.execute("UPDATE room_time SET time = %s WHERE room_name = %s", (current_time, room_name))
    return current_time


def add_isu(room_name: str, req_time: int, num_isu: int) -> bool:
    #print(f"add_isu(room_name={room_name}, req_time={req_time})")
    conn = connect_db()
    try:
        update_room_time(conn, room_name, req_time)
        cur = conn.cursor()
        cur.execute("INSERT INTO adding(room_name, time, isu) VALUES (%s, %s, '0') ON DUPLICATE KEY UPDATE isu=isu",
                    (room_name, req_time))

        cur.execute("SELECT isu FROM adding WHERE room_name = %s AND time = %s FOR UPDATE",
                    (room_name, req_time))
        isu = int(cur.fetchone()[0])
        isu += num_isu
        isu = str(isu)
        cur.execute("UPDATE adding SET isu=%s WHERE room_name=%s AND time=%s",
                    (isu, room_name, req_time))
    except Exception as e:
        conn.rollback()
        logging.exception("fail to add isu: room=%s time=%s isu=%s", room_name, req_time, num_isu)
        return False
    else:
        conn.commit()
        return True
    finally:
        conn.close()


def buy_item(room_name: str, req_time: int, item_id: int, count_bought: int) -> bool:
    #print(f"buy_item({room_name}, {req_time}, {item_id}, {count_bought})")
    conn = connect_db()
    try:
        update_room_time(conn, room_name, req_time)
        cur = conn.cursor()
        dcur = conn.cursor(MySQLdb.cursors.DictCursor)

        cur.execute("SELECT COUNT(*) FROM buying WHERE room_name = %s AND item_id = %s",
                    (room_name, item_id))
        count_buying, = cur.fetchone()
        if count_bought != count_buying:
            conn.rollback()
            logging.warn("item is already bought: room_name=%s, item_id=%s, count_bought=%s",
                         room_name, item_id, count_bought)
            return False

        total_milli_isu = 0

        cur.execute("SELECT isu FROM adding WHERE room_name = %s AND time <= %s",
                    (room_name, req_time))
        for (isu,) in cur:
            total_milli_isu += int(isu) * 1000

        cur.execute("SELECT item_id, ordinal, time FROM buying WHERE room_name = %s", (room_name,))
        buyings = cur.fetchall()
        for (buy_item_id, ordinal, item_time) in buyings:
            dcur.execute("SELECT * FROM m_item WHERE item_id=%s", (buy_item_id,))
            mitem = dcur.fetchone()
            cost = calc_item_price(mitem, ordinal)
            total_milli_isu -= cost * 1000
            if item_time < req_time:
                power = calc_item_power(mitem, ordinal)
                total_milli_isu += power * (req_time - item_time)

        dcur.execute("SELECT * FROM m_item WHERE item_id=%s", (item_id,))
        mitem = dcur.fetchone()
        cost = calc_item_price(mitem, count_bought+1) * 1000
        if total_milli_isu < cost:
            conn.rollback()
            logging.info("isu not enough")
            return False

        cur.execute("INSERT INTO buying(room_name, item_id, ordinal, time) VALUES(%s, %s, %s, %s)",
                    (room_name, item_id, count_bought+1, req_time))
    except Exception as e:
        conn.rollback()
        logging.exception("fail to buy item id=%s, bought=%d, time=%s", item_id, count_bought, req_time)
        return False
    else:
        conn.commit()
        return True
    finally:
        conn.close()


def get_current_time(conn) -> int:
    cur = conn.cursor()
    cur.execute("SELECT floor(unix_timestamp(current_timestamp(3))*1000)")
    t, = cur.fetchone()
    return t


def get_status(room_name: str) -> dict:
    conn = connect_db()
    try:
        current_time = update_room_time(conn, room_name, 0)

        dcur = conn.cursor(MySQLdb.cursors.DictCursor)
        dcur.execute("SELECT * FROM m_item")
        mitems = {m["item_id"]: m for m in dcur}
        dcur.close()

        cur = conn.cursor()
        cur.execute("SELECT time, isu FROM adding WHERE room_name=%s", (room_name,))
        addings = [Adding(t, i) for (t, i) in cur]

        cur.execute("SELECT item_id, ordinal, time FROM buying WHERE room_name=%s", (room_name,))
        buyings = [Buying(i, o, t) for (i, o, t) in cur]

        conn.commit()

        status = calc_status(current_time, mitems, addings, buyings)
        # calcStatusに時間がかかる可能性があるので タイムスタンプを取得し直す
        status = status._replace(time=get_current_time(conn))
        return status
    finally:
        conn.close()


async def serve(ws: 'aiohttp.web.WebSocketResponse', room_name: str):
    loop = asyncio.get_event_loop()

    status: dict = await loop.run_in_executor(None, get_status, room_name)
    last_status_time = time.time()
    await ws.send_json(status, dumps=simplejson.dumps)

    while not ws.closed:
        # 0.5 秒ごとに status を送る
        timeout = (last_status_time + 0.5) - time.time()
        if timeout < 0:
            status: dict = await loop.run_in_executor(None, get_status, room_name)
            last_status_time = time.time()
            await ws.send_json(status, dumps=simplejson.dumps)
            continue

        try:
            request: dict = await ws.receive_json(timeout=timeout)
        except asyncio.TimeoutError:
            continue

        print(f"received request: {request}")
        request_id: int = int(request["request_id"])
        action: str = str(request["action"])
        reqtime: int = int(request["time"])

        if action == "addIsu":
            # クライアントからは isu は文字列で送られてくる
            success = await loop.run_in_executor(None, add_isu, room_name, reqtime, int(request["isu"]))
        elif action == "buyItem":
            # count bought はその item_id がすでに買われている数.
            # count bought+1 個目を新たに買うことになる
            item_id = int(request["item_id"])
            count_bought = int(request["count_bought"])
            success = await loop.run_in_executor(None, buy_item, room_name, reqtime, item_id, count_bought)
        else:
            print(f"Invalid action: {action}")
            await ws.close()
            return

        if success:
            status = await loop.run_in_executor(None, get_status, room_name)
            last_status_time = time.time()
            await ws.send_json(status, dumps=simplejson.dumps)
        #else:
        #    print(f"fail: request={request}")

        await ws.send_json({
            "request_id": request_id,
            "is_success": success,
        })
