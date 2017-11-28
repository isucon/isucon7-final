from game import calc_status, calc_item_price, calc_item_power, int2exp, Schedule, Buying, Adding

def test_status_empty():
    """空の状態"""
    mitems = {}
    addings = []
    buyings = []

    status = calc_status(0, mitems, addings, buyings)

    assert status.adding == []
    assert len(status.schedule) == 1
    assert status.on_sale == []

    assert status.schedule[0].time == 0
    assert status.schedule[0].milli_isu == (0, 0)
    assert status.schedule[0].total_power == (0, 0)


def test_status_add():
    """add_isu を呼ぶ"""

    mitems = {}
    addings = [
        Adding(100, "1"),
        Adding(200, "2"),
        Adding(300, "1234567890123456789"),
    ]
    buyings = []

    s = calc_status(0, mitems, addings, buyings)

    assert len(s.adding) == 3
    assert len(s.schedule) == 4

    assert s.schedule[0].time == 0
    assert s.schedule[0].milli_isu == (0, 0)
    assert s.schedule[0].total_power == (0, 0)

    assert s.schedule[1].time == 100
    assert s.schedule[1].milli_isu == (1000, 0)
    assert s.schedule[1].total_power == (0, 0)

    assert s.schedule[2].time == 200
    assert s.schedule[2].milli_isu == (3000, 0)
    assert s.schedule[2].total_power == (0, 0)

    assert s.schedule[3].time == 300
    assert s.schedule[3].milli_isu == (123456789012345, 7)
    assert s.schedule[3].total_power == (0, 0)

    s = calc_status(500, mitems, addings, buyings)

    assert len(s.adding) == 0
    assert len(s.schedule) == 1

    assert s.schedule[0].time == 500
    assert s.schedule[0].milli_isu == (123456789012345, 7)
    assert s.schedule[0].total_power == (0, 0)


def test_status_buysingle():
    """1つ買う"""

    mitems = {1: {
        "item_id": 1,
        "power1": 0, "power2": 1, "power3": 0, "power4": 10,
        "price1": 0, "price2": 1, "price3": 0, "price4": 10,
    }}

    initial_isu = "10"

    addings = [
        Adding(0, initial_isu),
    ]
    buyings = [
        Buying(1, 1, 100),
    ]

    s = calc_status(0, mitems, addings, buyings)

    assert len(s.adding) == 0
    assert len(s.schedule) == 2
    assert len(s.items) == 1

    assert s.schedule[0].time == 0
    assert s.schedule[0].milli_isu == (0, 0)
    assert s.schedule[0].total_power == (0, 0)

    assert s.schedule[1].time == 100
    assert s.schedule[1].milli_isu == (0, 0)
    assert s.schedule[1].total_power == (10, 0)

def test_on_sale():
    """購入可能時刻のテスト"""

    mitems = {1: {
        "item_id": 1,
        "power1": 0, "power2": 1, "power3": 0, "power4": 1,  # (0*x+1)*(1**(0*x+1))
        "price1": 0, "price2": 1, "price3": 0, "price4": 1,
    }}
    addings = [
        Adding(0, "1"),
    ]
    buyings = [
        Buying(1, 1, 0),
    ]

    s = calc_status(1, mitems, addings, buyings)

    assert len(s.adding) == 0
    assert len(s.schedule) == 1
    assert len(s.on_sale) == 1
    assert len(s.items) == 1

    assert s.schedule[0].time == 1
    assert s.on_sale[0] == (1, 1000)

    assert s.items[0].count_bought == 1
    assert s.items[0].power == (1, 0)
    assert s.items[0].count_built == 1
    assert s.items[0].next_price == (1, 0)

def test_status_buy():
    """アイテム購入のテスト"""

    mitems = {
        1: {
        "item_id": 1,
        "power1": 1, "power2": 1, "power3": 3, "power4": 2,
        "price1": 1, "price2": 1, "price3": 7, "price4": 6,
        },
        2: {
        "item_id": 2,
        "power1": 1, "power2": 1, "power3": 7, "power4": 6,
        "price1": 1, "price2": 1, "price3": 3, "price4": 2,
        },
    }

    initial_isu = "10000000"

    addings = [
        Adding(0, initial_isu),
    ]
    buyings = [
        Buying(1, 1, 100),
        Buying(1, 2, 200),
        Buying(2, 1, 300),
        Buying(2, 2, 2001),
    ]

    s = calc_status(0, mitems, addings, buyings)

    assert len(s.adding) == 0
    assert len(s.schedule) == 4
    assert len(s.on_sale) == 2
    assert len(s.items) == 2

    total_power = 0

    milli_isu = int(initial_isu) * 1000
    milli_isu -= calc_item_price(mitems[1], 1) * 1000
    milli_isu -= calc_item_price(mitems[1], 2) * 1000
    milli_isu -= calc_item_price(mitems[2], 1) * 1000
    milli_isu -= calc_item_price(mitems[2], 2) * 1000

    # 0sec
    assert s.schedule[0].time == 0
    assert s.schedule[0].milli_isu == int2exp(milli_isu)
    assert s.schedule[0].total_power == int2exp(total_power)

    # 0.1sec
    total_power += calc_item_power(mitems[1], 1)
    assert s.schedule[1].time == 100
    assert s.schedule[1].milli_isu == int2exp(milli_isu)
    assert s.schedule[1].total_power == int2exp(total_power)

    # 0.2sec
    milli_isu += total_power * 100
    total_power += calc_item_power(mitems[1], 2)
    assert s.schedule[2].time == 200
    assert s.schedule[2].milli_isu == int2exp(milli_isu)
    assert s.schedule[2].total_power == int2exp(total_power)

    # 0.3sec
    milli_isu += total_power * 100
    total_power += calc_item_power(mitems[2], 1)
    assert s.schedule[3].time == 300
    assert s.schedule[3].milli_isu == int2exp(milli_isu)
    assert s.schedule[3].total_power == int2exp(total_power)

    # OnSale
    assert (1, 0) in s.on_sale
    assert (2, 0) in s.on_sale

def test_mitem():
    item = {
        "item_id": 1,
        "power1": 1, "power2": 2, "power3": 2, "power4": 3,
        "price1": 5, "price2": 4, "price3": 3, "price4": 2,
    }
    assert calc_item_power(item, 1) == 81
    assert calc_item_price(item, 1) == 2048

def test_conv():
    assert int2exp(int("0")) == (0, 0)
    assert int2exp(int("1234")) == (1234, 0)
    assert int2exp(int("11111111111111000000")) == (111111111111110, 5)

if __name__ == '__main__':
    test_status_empty()
    test_status_add()
    test_status_buysingle()
    test_on_sale()
    test_status_buy()
    test_mitem()
    test_conv()

