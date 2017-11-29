import asyncio
from collections import defaultdict, namedtuple
from functools import lru_cache
import logging
import os
import sys
import time

# namedtuple を dict として出力するために標準ライブラリの json ではなく
# simplejson を使います。
import simplejson


# types for JSON
Schedule = namedtuple("Schedule", ("time", "milli_isu", "total_power"))
Item = namedtuple("Item", ("item_id", "count_bought", "count_built", "next_price", "power", "building"))
OnSale = namedtuple("OnSale", ("item_id", "time"))
Building = namedtuple("Building", ("time", "count_built", "power"))
GameStatus = namedtuple("GameStatus", ("time", "adding", "schedule", "items", "on_sale"))
Adding = namedtuple("Adding", ("time", "isu"))
Buying = namedtuple("Buying", ("item_id", "ordinal", "time"))


class Room:
    def __init__(self, name):
        self.name = name
        self.time = 0
        self.status = None
        self.addings = []
        self.buyings = []


_rooms = {}  # TODO: 永続化

def get_room(name):
    try:
        return _rooms[name]
    except KeyError:
        r = Room(name)
        _rooms[name] = r
        return r


def initialize():
    _rooms.clear()


@lru_cache(100000)
def _calc_item_status(a, b, c, d, count):
    return (c * count + 1) * (d ** (a * count + b))


def calc_item_power(m: dict, count : int) -> int:
    """アイテムマスタ m から count 個目のそのアイテムの生産力を計算する"""
    a = m['power1']
    b = m['power2']
    c = m['power3']
    d = m['power4']
    return _calc_item_status(a, b, c, d, count)


def calc_item_price(m: dict, count : int) -> int:
    """アイテムマスタ m から count 個目のそのアイテムの価格を計算する"""
    a = m['price1']
    b = m['price2']
    c = m['price3']
    d = m['price4']
    return _calc_item_status(a, b, c, d, count)

# DBから吸い出したデータ
# item_id | power1 | power2 | power3 | power4 | price1 | price2 | price3 | price4 |
_m_items_dump = [
    ( 1,      0,     1,     0,  1,     0,    1,    1,  1),
    ( 2,      0,     1,     1,  1,     0,    1,    2,  1),
    ( 3,      1,    10,     0,  2,     1,    3,    1,  2),
    ( 4,      1,    24,     1,  2,     1,   10,    0,  3),
    ( 5,      1,    25,   100,  3,     2,   20,   20,  2),
    ( 6,      1,    30,   147, 13,     1,   22,   69, 17),
    ( 7,      5,    80,   128,  6,     6,   61,  200,  5),
    ( 8,     20,   340,   180,  3,     9,  105,  134, 14),
    ( 9,     55,   520,   335,  5,    48,  243,  600,  7),
    (10,    157,  1071,  1700, 12,   157,  625, 1000, 13),
    (11,   2000,  7500,  2600,  3,  2001, 5430, 1000,  3),
    (12,   1000,  9000,     0, 17,   963, 7689,    1, 19),
    (13,  11000, 11000, 11000, 23, 10000,    2,    2, 29),
]

_m_items = {}

for row in _m_items_dump:
    _m_items[row[0]] = {
        "item_id": row[0],
        "power1":  row[1],
        "power2":  row[2],
        "power3":  row[3],
        "power4":  row[4],
        "price1":  row[5],
        "price2":  row[6],
        "price3":  row[7],
        "price4":  row[8],
    }


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

    waiting_on_sale = []

    for item_id, m in mitems.items():
        item_power0[item_id] = int2exp(item_power[item_id])
        item_built0[item_id] = item_built[item_id]
        price = calc_item_price(m, item_bought[item_id]+1)
        item_price[item_id] = price
        if total_milli_isu >= price*1000:
            # 0 は 時刻 currentTime で購入可能であることを表す
            item_on_sale[item_id] = 0
        else:
            # on_sale 計算のための sale 待ちリスト
            waiting_on_sale.append((price*1000, item_id))

    # price が安い順にソートしておく
    waiting_on_sale.sort()

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
        bought = 0
        for i, (price, item_id) in enumerate(waiting_on_sale, 1):
            if price > total_milli_isu:
                break
            item_on_sale[item_id] = t
            bought = i
        del waiting_on_sale[:bought]

    gs_addings = [Adding(a.time, str(a.isu)) for a in adding_at.values()]

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


def update_room_time(room_name: str, req_time: int) -> int:
    room = get_room(room_name)
    current_time = get_current_time()

    if room.time > current_time:
        raise RuntimeError(f"room_time is future: room_time={room_time}, req_time={req_time}")

    if req_time and req_time < current_time:
        raise RuntimeError(f"req_time is past: req_time={req_time}, current_time={current_time}")

    room.time = current_time
    return current_time


def add_isu(room_name: str, req_time: int, num_isu: int) -> bool:
    #print(f"add_isu(room_name={room_name}, req_time={req_time})")
    try:
        update_room_time(room_name, req_time)
    except RuntimeError:
        logging.exception("fail to add isu: room=%s time=%s isu=%s", room_name, req_time, num_isu)
        return False

    room = get_room(room_name)
    for i, a in enumerate(room.addings):
        if a.time == req_time:
            a = a._replace(isu=a.isu + num_isu)
            room.addings[i] = a
            break
    else:
        room.addings.append(Adding(req_time, num_isu))

    return True


def buy_item(room_name: str, req_time: int, item_id: int, count_bought: int) -> bool:
    try:
        update_room_time(room_name, req_time)
    except RuntimeError:
        logging.exception("fail to buy isu")
        return False

    room = get_room(room_name)

    #TODO: カウンタを別に持つ
    count_buying = 0
    for b in room.buyings:
        if b.item_id == item_id:
            count_buying += 1

    if count_bought != count_buying:
        logging.warn("item is already bought: room_name=%s, item_id=%s, count_bought=%s",
                     room_name, item_id, count_bought)
        return False

    total_milli_isu = sum(a.isu for a in room.addings)
    total_milli_isu *= 1000

    buyings = room.buyings
    for (buy_item_id, ordinal, item_time) in buyings:
        cost = calc_item_price(_m_items[buy_item_id], ordinal)
        total_milli_isu -= cost * 1000
        if item_time < req_time:
            power = calc_item_power(_m_items[buy_item_id], ordinal)
            total_milli_isu += power * (req_time - item_time)

    cost = calc_item_price(_m_items[item_id], count_bought+1) * 1000
    if total_milli_isu < cost:
        logging.info("isu not enough")
        return False

    room.buyings.append(Buying(item_id, count_bought+1, req_time))
    return True


def get_current_time() -> int:
    return int(time.time() * 1000)


_status_cache = {}

def get_status(room_name: str, t0=None) -> dict:
    if t0 is None:
        t0 = get_current_time() - 200

    if room_name not in _status_cache or _status_cache[room_name][0] < t0:
        current_time = update_room_time(room_name, 0)
        room = get_room(room_name)
        status = calc_status(current_time, _m_items, room.addings, room.buyings)
        status = status._replace(time=get_current_time())
        status = simplejson.dumps(status)
        _status_cache[room_name] = (current_time, status)
    else:
        status = _status_cache[room_name][1]

    return status


async def serve(ws: 'aiohttp.web.WebSocketResponse', room_name: str):
    loop = asyncio.get_event_loop()

    status = get_status(room_name)
    last_status_time = time.time()
    await ws.send_str(status)

    while not ws.closed:
        # 0.5 秒ごとに status を送る
        timeout = (last_status_time + 0.5) - time.time()
        if timeout < 0:
            status = get_status(room_name)
            last_status_time = time.time()
            await ws.send_str(status)
            continue

        try:
            request: dict = await ws.receive_json(timeout=timeout)
        except asyncio.TimeoutError:
            continue

        #print(f"received request: {request}")
        request_id: int = int(request["request_id"])
        action: str = str(request["action"])
        reqtime: int = int(request["time"])

        if action == "addIsu":
            # クライアントからは isu は文字列で送られてくる
            success = add_isu(room_name, reqtime, int(request['isu']))
        elif action == "buyItem":
            # count bought はその item_id がすでに買われている数.
            # count bought+1 個目を新たに買うことになる
            item_id = int(request["item_id"])
            count_bought = int(request["count_bought"])
            success = buy_item(room_name, reqtime, item_id, count_bought)
        else:
            print(f"Invalid action: {action}")
            await ws.close()
            return

        if success:
            t = get_current_time()
            await asyncio.sleep(0.2)  # get_status() を他のリクエストとまとめる
            status = get_status(room_name, t)
            last_status_time = time.time()
            await ws.send_str(status)
        #else:
        #    print(f"fail: request={request}")

        await ws.send_json({
            "request_id": request_id,
            "is_success": success,
        })
