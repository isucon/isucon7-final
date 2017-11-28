const bigint = require('bigint')
const MItem = require('./MItem')
const Exponential = require('./Exponential')

class Game {
  constructor(roomName, pool) {
    this.roomName = roomName
    this.pool = pool
  }

  async getStatus () {
    const connection = await this.pool.getConnection()
    await connection.beginTransaction()

    try {
      const currentTime = await this.updateRoomTime(connection, 0)
      const mItems = {}
      const [items] = await connection.query('SELECT * FROM m_item')
      for (let item of items) {
        mItems[item.item_id] = new MItem(item)
      }
      const [addings] = await connection.query('SELECT time, isu FROM adding WHERE room_name = ?', [this.roomName])
      const [buyings] = await connection.query('SELECT item_id, ordinal, time FROM buying WHERE room_name = ?', [this.roomName])
      await connection.commit()
      connection.release()

      const status = this.calcStatus(currentTime, mItems, addings, buyings)

      // calcStatusに時間がかかる可能性があるので タイムスタンプを取得し直す
      const latestTime = await this.getCurrentTime()
      status.time = latestTime

      return status

    } catch (e) {
      await connection.rollback()
      connection.release()
      throw e
    }
  }

  async addIsu (reqIsu, reqTime) {
    try {
      const connection = await this.pool.getConnection()
      await connection.beginTransaction()

      try {
        await this.updateRoomTime(connection, reqTime)
        await connection.query('INSERT INTO adding(room_name, time, isu) VALUES (?, ?, \'0\') ON DUPLICATE KEY UPDATE isu=isu', [this.roomName, reqTime])

        const [[{ isu }]] = await connection.query('SELECT isu FROM adding WHERE room_name = ? AND time = ? FOR UPDATE', [this.roomName, reqTime])
        const newIsu = reqIsu.add(bigint(isu))

        await connection.query('UPDATE adding SET isu = ? WHERE room_name = ? AND time = ?', [newIsu.toString(), this.roomName, reqTime])
        await connection.commit()
        connection.release()
        return true

      } catch (e) {
        await connection.rollback()
        connection.release()
        throw e
      }
    } catch (e) {
      console.error(e)
      return false
    }
  }

  async buyItem (itemId, countBought, reqTime) {
    try {
      const connection = await this.pool.getConnection()
      await connection.beginTransaction()

      try {
        await this.updateRoomTime(connection, reqTime)
        const [[{ countBuying }]] = await connection.query('SELECT COUNT(*) as countBuying FROM buying WHERE room_name = ? AND item_id = ?', [this.roomName, itemId])
        if (parseInt(countBuying, 10) != countBought) {
          throw new Error(`roomName=${this.roomName}, itemId=${itemId} countBought+1=${countBought+1} is already bought`)
        }

        let totalMilliIsu = bigint('0')
        const [addings] = await connection.query('SELECT isu FROM adding WHERE room_name = ? AND time <= ?', [this.roomName, reqTime])
        for (let { isu } of addings) {
          totalMilliIsu = totalMilliIsu.add(bigint(isu).mul(bigint('1000')))
        }

        const [buyings] = await connection.query('SELECT item_id, ordinal, time FROM buying WHERE room_name = ?', [this.roomName])
        for (let b of buyings) {
          let [[mItem]] = await connection.query('SELECT * FROM m_item WHERE item_id = ?', [b.item_id])
          let item = new MItem(mItem)
          let cost = item.getPrice(parseInt(b.ordinal, 10)).mul(bigint('1000'))
          totalMilliIsu = totalMilliIsu.sub(cost)
          if (parseInt(b.time, 10) <= reqTime) {
            let gain = item.getPower(parseInt(b.ordinal, 10)).mul(bigint('' + (reqTime - parseInt(b.time, 10))))
            totalMilliIsu = totalMilliIsu.add(gain)
          }
        }

        const [[mItem]] = await connection.query('SELECT * FROM m_item WHERE item_id = ?', [itemId])
        const item = new MItem(mItem)
        const need = item.getPrice(countBought + 1).mul(bigint('1000'))
        if (totalMilliIsu.cmp(need) < 0) {
          throw new Error('not enough')
        }

        await connection.query('INSERT INTO buying(room_name, item_id, ordinal, time) VALUES(?, ?, ?, ?)', [this.roomName, itemId, countBought + 1, reqTime])
        await connection.commit()
        connection.release()
        return true

      } catch (e) {
        await connection.rollback()
        connection.release()
        throw e
      }
    } catch (e) {
      console.error(e)
      return false
    }
  }

  // 部屋のロックを取りタイムスタンプを更新する
  //
  // トランザクション開始後この関数を呼ぶ前にクエリを投げると、
  // そのトランザクション中の通常のSELECTクエリが返す結果がロック取得前の
  // 状態になることに注意 (keyword: MVCC, repeatable read).
  async updateRoomTime (connection, reqTime) {
    // See page 13 and 17 in https://www.slideshare.net/ichirin2501/insert-51938787
    await connection.query('INSERT INTO room_time(room_name, time) VALUES (?, 0) ON DUPLICATE KEY UPDATE time = time', [this.roomName])
    const [[{ time }]] = await connection.query('SELECT time FROM room_time WHERE room_name = ? FOR UPDATE', [this.roomName])
    const [[{ currentTime }]] = await connection.query('SELECT floor(unix_timestamp(current_timestamp(3))*1000) AS currentTime')
    if (parseInt(time, 10) > parseInt(currentTime, 10)) {
      throw new Error('room time is future')
    }
    if (reqTime !== 0) {
      if (reqTime < parseInt(currentTime, 10)) {
        throw new Error('reqTime is past')
      }
    }

    await connection.query('UPDATE room_time SET time = ? WHERE room_name = ?', [currentTime, this.roomName])
    return currentTime
  }

  calcStatus (currentTime, mItems, addings, buyings) {
    // 1ミリ秒に生産できる椅子の単位をミリ椅子とする
    let totalMilliIsu = bigint('0')
    let totalPower    = bigint('0')

    const itemPower    = {} // ItemID => Power
    const itemPrice    = {} // ItemID => Price
    const itemOnSale   = {} // ItemID => OnSale
    const itemBuilt    = {} // ItemID => BuiltCount
    const itemBought   = {} // ItemID => CountBought
    const itemBuilding = {} // ItemID => Buildings
    const itemPower0   = {} // ItemID => currentTime における Power
    const itemBuilt0   = {} // ItemID => currentTime における BuiltCount

    const addingAt = {} // Time => currentTime より先の Adding
    const buyingAt = {} // Time => currentTime より先の Buying

    for (let itemId in mItems) {
      itemPower[itemId] = bigint('0')
      itemBuilding[itemId] = []
    }

    for (let a of addings) {
      // adding は adding.time に isu を増加させる
      if (a.time <= currentTime) {
        totalMilliIsu = totalMilliIsu.add(bigint(a.isu).mul(bigint('1000')))
      } else {
        addingAt[a.time] = a
      }
    }

    for (let b of buyings) {
      // buying は 即座に isu を消費し buying.time からアイテムの効果を発揮する
      itemBought[b.item_id] = itemBought[b.item_id] ? itemBought[b.item_id] + 1 : 1
      const m = mItems[b.item_id]
      totalMilliIsu = totalMilliIsu.sub(m.getPrice(b.ordinal).mul(bigint('1000')))

      if (b.time <= currentTime) {
        itemBuilt[b.item_id] = itemBuilt[b.item_id] ? itemBuilt[b.item_id] + 1 : 1
        const power = m.getPower(itemBought[b.item_id])
        totalMilliIsu = totalMilliIsu.add(power.mul(bigint(currentTime - b.time)))
        totalPower = totalPower.add(power)
        itemPower[b.item_id] = itemPower[b.item_id].add(power)
      } else {
        buyingAt[b.time] = buyingAt[b.time] || []
        buyingAt[b.time].push(b)
      }
    }

    for (let itemId in mItems) {
      const m = mItems[itemId]
      itemPower0[m.itemId] = this.big2exp(itemPower[m.itemId])
      itemBuilt0[m.itemId] = itemBuilt[m.itemId]
      const price = m.getPrice((itemBought[m.itemId] || 0) + 1)
      itemPrice[m.itemId] = price
      if (0 <= totalMilliIsu.cmp(price.mul(bigint('1000')))) {
        itemOnSale[m.itemId] = 0 // 0 は 時刻 currentTime で購入可能であることを表す
      }
    }

    const schedule = [
      {
        time:        currentTime,
        milli_isu:   this.big2exp(totalMilliIsu),
        total_power: this.big2exp(totalPower),
      }
    ]

    // currentTime から 1000 ミリ秒先までシミュレーションする
    for (let t = currentTime + 1; t <= currentTime + 1000; t++) {
      totalMilliIsu = totalMilliIsu.add(totalPower)
      let updated = false

      // 時刻 t で発生する adding を計算する
      if (addingAt[t]) {
        let a = addingAt[t]
        updated = true
        totalMilliIsu = totalMilliIsu.add(bigint(a.isu).mul(bigint('1000')))
      }

      // 時刻 t で発生する buying を計算する
      if (buyingAt[t]) {
        updated = true
        const updatedID = {}
        for (let b of buyingAt[t]) {
          const m = mItems[b.item_id]
          updatedID[b.item_id] = true
          itemBuilt[b.item_id] = itemBuilt[b.item_id] ? itemBuilt[b.item_id] + 1 : 1
          const power = m.getPower(b.ordinal)
          itemPower[b.item_id] = itemPower[b.item_id].add(power)
          totalPower = totalPower.add(power)
        }
        for (let id in updatedID) {
          itemBuilding[id].push({
            time:        t,
            count_built: itemBuilt[id],
            power:       this.big2exp(itemPower[id]),
          })
        }
      }

      if (updated) {
        schedule.push({
          time:        t,
          milli_isu:   this.big2exp(totalMilliIsu),
          total_power: this.big2exp(totalPower),
        })
      }

      // 時刻 t で購入可能になったアイテムを記録する
      for (let itemId in mItems) {
        if (typeof itemOnSale[itemId] !== 'undefined') {
          continue;
        }
        if (0 <= totalMilliIsu.cmp(itemPrice[itemId].mul(bigint('1000')))) {
          itemOnSale[itemId] = t
        }
      }
    }

    const gsAdding = []
    for (let a of Object.values(addingAt)) {
      gsAdding.push(a)
    }

    const gsItems = []
    for (let itemId in mItems) {
      gsItems.push({
        item_id:      parseInt(itemId, 10),
        count_bought: itemBought[itemId] || 0,
        count_built:  itemBuilt0[itemId] || 0,
        next_price:   this.big2exp(itemPrice[itemId]),
        power:        itemPower0[itemId],
        building:     itemBuilding[itemId],
      })
    }

    const gsOnSale = []
    for (let itemId in itemOnSale) {
      let t = itemOnSale[itemId]
      gsOnSale.push({
        item_id: parseInt(itemId, 10),
        time:    t,
      })
    }

    return {
      time:      0,
      adding:    gsAdding,
      schedule:  schedule,
      items:     gsItems,
      on_sale:   gsOnSale,
    }
  }

  async getCurrentTime () {
    try {
      const [[{currentTime}]] = await this.pool.query('SELECT floor(unix_timestamp(current_timestamp(3))*1000) AS currentTime')
      return parseInt(currentTime, 10)
    } catch (e) {
      console.error(e)
      return 0
    }
  }

  big2exp (n) {
    const s = n.toString()
    if (s.length <= 15) {
      return new Exponential({
        mantissa: n.toNumber(),
        exponent: 0,
      })
    }

    const t = parseInt(s.slice(0, 15), 10)
    return new Exponential({
      mantissa: t,
      exponent: s.length - 15,
    })
  }
}

module.exports = Game
