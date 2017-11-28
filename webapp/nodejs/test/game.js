const assert = require('assert')
const bigint = require('bigint')
const Game = require('../Game')
const Exponential = require('../Exponential')
const MItem = require('../MItem')

describe('Game', () => {
  it('TestStatusEmpty', () => {
    const game = new Game('xxx', null)

    const mItems = {}
    const addings = []
    const buyings = []

    const s = game.calcStatus(0, mItems, addings, buyings)

    assert.equal(0, s.adding.length)
    assert.equal(1, s.schedule.length)
    assert.equal(0, s.on_sale.length)

    assert.equal(0, s.schedule[0].time)
    assert.deepEqual(new Exponential({ mantissa: 0, exponent: 0 }), s.schedule[0].milli_isu)
    assert.deepEqual(new Exponential({ mantissa: 0, exponent: 0 }), s.schedule[0].total_power)
  })

  it('TestStatusAdd', () => {
    const game = new Game('xxx', null)

    const mItems = {}
    const addings = [
      { time: 100, isu: '1' },
      { time: 200, isu: '2' },
      { time: 300, isu: '1234567890123456789' },
    ]
    const buyings = []

    let s = game.calcStatus(0, mItems, addings, buyings)
    assert.equal(s.adding.length, 3)
    assert.equal(s.schedule.length, 4)

    assert.equal(0, s.schedule[0].time)
    assert.deepEqual(new Exponential({ mantissa: 0, exponent: 0 }), s.schedule[0].milli_isu)
    assert.deepEqual(new Exponential({ mantissa: 0, exponent: 0 }), s.schedule[0].total_power)

    assert.equal(100, s.schedule[1].time)
    assert.deepEqual(new Exponential({ mantissa: 1000, exponent: 0 }), s.schedule[1].milli_isu)
    assert.deepEqual(new Exponential({ mantissa: 0, exponent: 0 }), s.schedule[1].total_power)

    assert.equal(200, s.schedule[2].time)
    assert.deepEqual(new Exponential({ mantissa: 3000, exponent: 0 }), s.schedule[2].milli_isu)
    assert.deepEqual(new Exponential({ mantissa: 0, exponent: 0 }), s.schedule[2].total_power)

    assert.equal(300, s.schedule[3].time)
    assert.deepEqual(new Exponential({ mantissa: 123456789012345, exponent: 7 }), s.schedule[3].milli_isu)
    assert.deepEqual(new Exponential({ mantissa: 0, exponent: 0 }), s.schedule[3].total_power)

    s = game.calcStatus(500, mItems, addings, buyings)
    assert.equal(0, s.adding.length)
    assert.equal(1, s.schedule.length)

    assert.equal(500, s.schedule[0].time)
    assert.deepEqual(new Exponential({ mantissa: 123456789012345, exponent: 7 }), s.schedule[0].milli_isu)
    assert.deepEqual(new Exponential({ mantissa: 0, exponent: 0 }), s.schedule[0].total_power)
  })

  it('TestStatusBuySingle', () => {
    const game = new Game('xxx', null)
    const x = new MItem({
      item_id: 1,
      power1: 0,
      power2: 1,
      power3: 0,
      power4: 10,
      price1: 0,
      price2: 1,
      price3: 0,
      price4: 10,
    })
    const mItems = { 1: x }
    const initialIsu = "10"
    const addings = [
      { time: 0, isu: initialIsu },
    ]
    const buyings = [
      { item_id: 1, ordinal: 1, time: 100 },
    ]

    const s = game.calcStatus(0, mItems, addings, buyings)
    assert.equal(0, s.adding.length)
    assert.equal(2, s.schedule.length)
    assert.equal(1, s.items.length)

    assert.equal(0, s.schedule[0].time)
    assert.deepEqual(new Exponential({ mantissa: 0, exponent: 0 }), s.schedule[0].milli_isu)
    assert.deepEqual(new Exponential({ mantissa: 0, exponent: 0 }), s.schedule[0].total_power)

    assert.equal(100, s.schedule[1].time)
    assert.deepEqual(new Exponential({ mantissa: 0, exponent: 0 }), s.schedule[1].milli_isu)
    assert.deepEqual(new Exponential({ mantissa: 10,exponent:  0 }), s.schedule[1].total_power)
  })

  it('TestOnSale', () => {
    const game = new Game('xxx', null)
    const x = new MItem({
      item_id: 1,
      power1: 0, power2: 1, power3: 0, power4: 1, // power: (0x+1)*1^(0x+1)
      price1: 0, price2: 1, price3: 0, price4: 1, // price: (0x+1)*1^(0x+1)
    })
    const mItems = { 1: x }
    const addings = [{ time: 0, isu: "1" }]
    const buyings = [{ item_id: 1, ordinal: 1, time: 0 }]

    const s = game.calcStatus(1, mItems, addings, buyings)
    assert.equal(0, s.adding.length)
    assert.equal(1, s.schedule.length)
    assert.equal(1, s.on_sale.length)
    assert.equal(1, s.items.length)

    assert.equal(1, s.schedule[0].time)
    assert.deepEqual({ item_id: 1, time: 1000 }, s.on_sale[0])

    assert.equal(1, s.items[0].count_bought)
    assert.deepEqual(new Exponential({ mantissa: 1, exponent: 0 }), s.items[0].power)
    assert.equal(1, s.items[0].count_built)
    assert.deepEqual(new Exponential({ mantissa: 1, exponent: 0 }), s.items[0].next_price)
  })

  it('TestStatusBuy', () => {
    const game = new Game('xxx', null)
    const x = new MItem({
      item_id: 1,
      power1: 1,
      power2: 1,
      power3: 3,
      power4: 2,
      price1: 1,
      price2: 1,
      price3: 7,
      price4: 6,
    })
    const y = new MItem({
      item_id: 2,
      price1: 1,
      price2: 1,
      price3: 3,
      price4: 2,
      power1: 1,
      power2: 1,
      power3: 7,
      power4: 6,
    })
    const mItems = { 1: x, 2: y }
    const initialIsu = "10000000"
    const addings = [
      { time: 0, isu: initialIsu },
    ]
    const buyings = [
      { item_id: 1, ordinal: 1, time: 100 },
      { item_id: 1, ordinal: 2, time: 200 },
      { item_id: 2, ordinal: 1, time: 300 },
      { item_id: 2, ordinal: 2, time: 2001 },
    ]

    const s = game.calcStatus(0, mItems, addings, buyings)
    assert.equal(0, s.adding.length)
    assert.equal(4, s.schedule.length)
    assert.equal(2, s.on_sale.length)
    assert.equal(2, s.items.length)

    let totalPower = bigint('0')
    let milliIsu = bigint(initialIsu).mul(bigint('1000'))
    milliIsu = milliIsu.sub(x.getPrice(1).mul(bigint('1000')))
    milliIsu = milliIsu.sub(x.getPrice(2).mul(bigint('1000')))
    milliIsu = milliIsu.sub(y.getPrice(1).mul(bigint('1000')))
    milliIsu = milliIsu.sub(y.getPrice(2).mul(bigint('1000')))

    // 0sec
    assert.equal(0, s.schedule[0].time)
    assert.deepEqual(game.big2exp(milliIsu), s.schedule[0].milli_isu)
    assert.deepEqual(game.big2exp(totalPower), s.schedule[0].total_power)

    // 0.1sec
    totalPower = totalPower.add(x.getPower(1))
    assert.equal(100, s.schedule[1].time)
    assert.deepEqual(game.big2exp(milliIsu), s.schedule[1].milli_isu)
    assert.deepEqual(game.big2exp(totalPower), s.schedule[1].total_power)

    // 0.2sec
    milliIsu = milliIsu.add(totalPower.mul(bigint('100')))
    totalPower = totalPower.add(x.getPower(2))
    assert.equal(200, s.schedule[2].time)
    assert.deepEqual(game.big2exp(milliIsu), s.schedule[2].milli_isu)
    assert.deepEqual(game.big2exp(totalPower), s.schedule[2].total_power)

    // 0.3sec
    milliIsu = milliIsu.add(totalPower.mul(bigint('100')))
    totalPower = totalPower.add(y.getPower(1))
    assert.equal(300, s.schedule[3].time)
    assert.deepEqual(game.big2exp(milliIsu), s.schedule[3].milli_isu)
    assert.deepEqual(game.big2exp(totalPower), s.schedule[3].total_power)

    // on_sale
    assert.ok(s.on_sale.some(s => s.item_id === 1 && s.time === 0))
    assert.ok(s.on_sale.some(s => s.item_id === 2 && s.time === 0))
  })

  it('TestMItem', () => {
    const item = new MItem({
      item_id: 1,
      power1: 1,
      power2: 2,
      power3: 2,
      power4: 3,
      price1: 5,
      price2: 4,
      price3: 3,
      price4: 2,
    })
    assert.equal(item.getPower(1).cmp(bigint('81')), 0)
    assert.equal(item.getPrice(1).cmp(bigint('2048')), 0)
  })

  it('TestConv', () => {
    const game = new Game('xxx', null)
    assert.deepEqual(new Exponential({ mantissa: 0, exponent: 0 }), game.big2exp(bigint("0")))
    assert.deepEqual(new Exponential({ mantissa: 1234, exponent: 0 }), game.big2exp(bigint("1234")))
    assert.deepEqual(new Exponential({ mantissa: 111111111111110, exponent: 5 }), game.big2exp(bigint("11111111111111000000")))
  })
})
