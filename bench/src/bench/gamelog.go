package main

import (
	"compress/gzip"
	"context"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"sort"
	"sync"
	"time"
)

var (
	gamelogMtx sync.Mutex
	gamelog    = map[string]*gameLogger{}

	saveGameLogDump int
)

type itemMasterElement struct {
	v1K *big.Int
	exp Exponential
}

type itemMaster struct {
	cache map[int]map[int]*itemMasterElement
	power map[int]map[int]*big.Int
}

func newItemMaster() *itemMaster {
	r := new(itemMaster)
	r.cache = make(map[int]map[int]*itemMasterElement)
	r.power = make(map[int]map[int]*big.Int)
	return r
}

func (obj *itemMaster) getData(itemID, count int) *itemMasterElement {
	if a, ok1 := obj.cache[itemID]; ok1 {
		if v, ok2 := a[count]; ok2 {
			return v
		}
	} else {
		obj.cache[itemID] = make(map[int]*itemMasterElement)
	}

	m := mItems[itemID]
	price := m.GetPrice(count)
	r := &itemMasterElement{
		v1K: new(big.Int).Mul(price, big.NewInt(1000)),
		exp: big2exp(price),
	}
	obj.cache[itemID][count] = r
	return r
}

func (obj *itemMaster) getPower(itemID, count int) *big.Int {
	if a, ok1 := obj.power[itemID]; ok1 {
		if v, ok2 := a[count]; ok2 {
			return v
		}
	} else {
		obj.power[itemID] = make(map[int]*big.Int)
	}

	m := mItems[itemID]
	r := m.GetPower(count)
	obj.power[itemID][count] = r
	return r
}

func getGameLogger(room string) *gameLogger {
	gamelogMtx.Lock()
	g, ok := gamelog[room]
	if !ok {
		g = new(gameLogger)
		gamelog[room] = g
	}
	gamelogMtx.Unlock()

	return g
}

type gameLogger struct {
	mtx      sync.Mutex
	status   []*GameStatusLog
	request  []*GameRequestLog
	response []*GameResponseLog
}

type gameLogDump struct {
	Room        string
	IsPreTest   bool
	StatusLog   []*GameStatusLog
	RequestLog  []*GameRequestLog
	ResponseLog []*GameResponseLog
}

func logOnStatus(room string, s *GameStatusLog) {
	g := getGameLogger(room)

	g.mtx.Lock()
	g.status = append(g.status, s)
	g.mtx.Unlock()
}

func logOnRequest(room string, req *GameRequestLog) {
	g := getGameLogger(room)

	g.mtx.Lock()
	g.request = append(g.request, req)
	g.mtx.Unlock()
}

func logOnResponse(room string, res *GameResponseLog) {
	g := getGameLogger(room)

	g.mtx.Lock()
	g.response = append(g.response, res)
	g.mtx.Unlock()
}

func getLatestStatus(statusList []*GameStatusLog, t1 time.Time, t2 time.Time) *GameStatusLog {
	idx := -1
	var t int64
	for i, x := range statusList {
		// t1 <= x.ClientTime <= t2
		if x.ClientTime.Sub(t1) >= 0 && t2.Sub(x.ClientTime) >= 0 {
			if idx < 0 || x.Schedule[0].Time >= t {
				t = x.Schedule[0].Time
				idx = i
			}
		}
	}
	if idx < 0 {
		return nil
	}
	return statusList[idx]
}

func validateAddIsu(requestID int, time int64, isu string, status *GameStatusLog) error {
	t0 := status.Schedule[0].Time
	if time > t0 {
		ok := false
		for _, x := range status.Adding {
			if x.Time == time {
				if str2big(x.Isu).Cmp(str2big(isu)) < 0 {
					return fmt.Errorf("addIsu (request_id = %v) に対するaddingの反映が行われていません : addingの time = %v の要素のisuの値が正しくありません : actual %v, expected %v 以上", requestID, time, x.Isu, isu)
				}
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("addIsu (request_id = %v) に対するaddingの反映が行われていません : addingに time = %v の要素が存在しません", requestID, time)
		}
		idx := -1
		for i, x := range status.Schedule {
			if x.Time == time {
				idx = i
			}
		}
		if idx <= 0 {
			return fmt.Errorf("addIsu (request_id = %v) に対するscheduleの反映が行われていません : scheduleに time = %v の要素が存在しません", requestID, time)
		}
	}

	return nil
}

func validateBuyItem(requestID int, itemID int, countBought int, status *GameStatusLog) error {
	for _, x := range status.Items {
		if x.ItemID == itemID {
			if x.CountBought < countBought {
				return fmt.Errorf("buyItem (request_id = %v) に対するitems[item_id = %v].count_boughtの反映が行われていません : actual %v, expected %v 以上",
					requestID, itemID, x.CountBought, countBought)
			}
		}
	}

	return nil
}

func validateStatus(status *GameStatusLog, addIsuDict map[int64]*big.Int, buyItemDict map[int]map[int]int64, itemMasterObj *itemMaster) error {
	var (
		// 1ミリ秒に生産できる椅子の単位をミリ椅子とする
		totalMilliIsu = big.NewInt(0)
		totalPower    = big.NewInt(0)

		itemPrice = map[int]*itemMasterElement{}

		itemPower    = map[int]*big.Int{}    // ItemID => Power
		itemOnSale   = map[int]int64{}       // ItemID => OnSale
		itemBuilt    = map[int]int{}         // ItemID => BuiltCount
		itemBought   = map[int]int{}         // ItemID => CountBought
		itemBuilding = map[int][]Building{}  // ItemID => Buildings
		itemPower0   = map[int]Exponential{} // ItemID => currentTime における Power
		itemBuilt0   = map[int]int{}         // ItemID => currentTime における BuiltCount

		adding   = map[int64]*big.Int{}
		buyingAt = map[int64][]Buying{}
	)

	currentTime := status.Schedule[0].Time

	for itemID := range mItems {
		itemPower[itemID] = big.NewInt(0)
		itemBuilding[itemID] = []Building{}
	}

	for t, v := range addIsuDict {
		if t <= currentTime {
			totalMilliIsu.Add(totalMilliIsu, new(big.Int).Mul(v, big.NewInt(1000)))
		}
	}

	for i, a := range status.Adding {
		adding[a.Time] = str2big(a.Isu)
		v, ok := addIsuDict[a.Time]
		if !ok {
			return fmt.Errorf("adding[%v].time が正しくありません : time = %v のaddIsuが存在しません", i, a.Time)
		} else if adding[a.Time].Cmp(v) > 0 {
			return fmt.Errorf("adding[%v].isu が正しくありません : actual %v, expected %v 以下", i, a.Isu, v.String())
		}
	}

	for _, b := range status.Items {
		a := buyItemDict[b.ItemID]
		itemBuilt[b.ItemID] = b.CountBuilt
		itemBought[b.ItemID] = b.CountBought
		for i := 1; i <= b.CountBought; i++ {
			t, ok := a[i-1]
			if !ok {
				return fmt.Errorf("items[item_id = %v].count_bought が正しくありません または count_bought = %v のbuyItemが存在しません : actual %v, expected %v 以下",
					b.ItemID, i-1, b.CountBought, i-1)
			}
			totalMilliIsu.Sub(totalMilliIsu, itemMasterObj.getData(b.ItemID, i).v1K)
			if i <= b.CountBuilt {
				if t > currentTime {
					return fmt.Errorf("items[item_id = %v].count_built が正しくありません : actual %v, expected %v 以下",
						b.ItemID, b.CountBuilt, i-1)
				}
				power := itemMasterObj.getPower(b.ItemID, i)
				totalMilliIsu.Add(totalMilliIsu, new(big.Int).Mul(power, big.NewInt(currentTime-t)))
				totalPower.Add(totalPower, power)
				itemPower[b.ItemID].Add(itemPower[b.ItemID], power)
			} else {
				if t <= currentTime {
					return fmt.Errorf("items[item_id = %v].count_built が正しくありません : actual %v, expected %v 以上",
						b.ItemID, b.CountBuilt, i)
				}
				buyingAt[t] = append(buyingAt[t], Buying{
					ItemID:  b.ItemID,
					Ordinal: i,
					Time:    t,
				})
			}
		}
		if t, ok := a[b.CountBought]; ok {
			if t <= currentTime {
				return fmt.Errorf("items[item_id = %v].count_bought が正しくありません : actual %v, expected %v 以上",
					b.ItemID, b.CountBought, b.CountBought+1)
			}
		}
	}

	for itemID := range mItems {
		itemPower0[itemID] = big2exp(itemPower[itemID])
		itemBuilt0[itemID] = itemBuilt[itemID]
		itemPrice[itemID] = itemMasterObj.getData(itemID, itemBought[itemID]+1)
		if 0 <= totalMilliIsu.Cmp(itemPrice[itemID].v1K) {
			itemOnSale[itemID] = 0 // 0 は 時刻 currentTime で購入可能であることを表す
		}
	}

	if totalMilliIsu.Sign() < 0 {
		return fmt.Errorf("時刻 %v で椅子の数が負になります", currentTime)
	}

	var schedule []Schedule
	schedule = append(schedule, Schedule{
		Time:       currentTime,
		MilliIsu:   big2exp(totalMilliIsu),
		TotalPower: big2exp(totalPower),
	})

	// currentTime から 1000 ミリ秒先までシミュレーションする
	for t := currentTime + 1; t <= currentTime+1000; t++ {
		totalMilliIsu.Add(totalMilliIsu, totalPower)
		updated := false

		// 時刻 t で発生する adding を計算する
		if v, ok := adding[t]; ok {
			updated = true
			totalMilliIsu.Add(totalMilliIsu, new(big.Int).Mul(v, big.NewInt(1000)))
		}

		// 時刻 t で発生する buying を計算する
		if _, ok := buyingAt[t]; ok {
			updated = true
			updatedID := map[int]bool{}
			for _, b := range buyingAt[t] {
				updatedID[b.ItemID] = true
				power := itemMasterObj.getPower(b.ItemID, b.Ordinal)
				itemPower[b.ItemID].Add(itemPower[b.ItemID], power)
				itemBuilt[b.ItemID]++
				totalPower.Add(totalPower, power)
			}
			for id := range updatedID {
				itemBuilding[id] = append(itemBuilding[id], Building{
					Time:       t,
					CountBuilt: itemBuilt[id],
					Power:      big2exp(itemPower[id]),
				})
			}
		}

		if updated {
			schedule = append(schedule, Schedule{
				Time:       t,
				MilliIsu:   big2exp(totalMilliIsu),
				TotalPower: big2exp(totalPower),
			})
		}

		// 時刻 t で購入可能になったアイテムを記録する
		for itemID := range mItems {
			if _, ok := itemOnSale[itemID]; ok {
				continue
			}
			if 0 <= totalMilliIsu.Cmp(itemPrice[itemID].v1K) {
				itemOnSale[itemID] = t
			}
		}
	}

	if len(status.Schedule) != len(schedule) {
		return fmt.Errorf("schedule の要素数が正しくありません : actual %v, expected %v", len(status.Schedule), len(schedule))
	}
	for i, x := range status.Schedule {
		if x.Time != schedule[i].Time {
			return fmt.Errorf("schedule[%v].time が正しくありません : actual %v, expected %v", i, x.Time, schedule[i].Time)
		}
		if x.MilliIsu != schedule[i].MilliIsu {
			return fmt.Errorf("schedule[%v].milli_isu が正しくありません : actual %v, expected %v", i, x.MilliIsu, schedule[i].MilliIsu)
		}
		if x.TotalPower != schedule[i].TotalPower {
			return fmt.Errorf("schedule[%v].total_power が正しくありません : actual %v, expected %v", i, x.TotalPower, schedule[i].TotalPower)
		}
	}

	for _, x := range status.Items {
		if x.CountBought != itemBought[x.ItemID] {
			return fmt.Errorf("items[item_id = %v].count_bought が正しくありません : actual %v, expected %v", x.ItemID, x.CountBought, itemBought[x.ItemID])
		}
		if x.CountBuilt != itemBuilt0[x.ItemID] {
			return fmt.Errorf("items[item_id = %v].count_built が正しくありません : actual %v, expected %v", x.ItemID, x.CountBuilt, itemBuilt0[x.ItemID])
		}
		if x.NextPrice != itemPrice[x.ItemID].exp {
			return fmt.Errorf("items[item_id = %v].next_price が正しくありません : actual %v, expected %v", x.ItemID, x.NextPrice, itemPrice[x.ItemID].exp)
		}
		if x.Power != itemPower0[x.ItemID] {
			return fmt.Errorf("items[item_id = %v].power が正しくありません : actual %v, expected %v", x.ItemID, x.Power, itemPower0[x.ItemID])
		}
		if len(x.Building) != len(itemBuilding[x.ItemID]) {
			return fmt.Errorf("items[item_id = %v].building の要素数が正しくありません : actual %v, expected %v", x.ItemID, len(x.Building), len(itemBuilding[x.ItemID]))
		}
		for i, y := range x.Building {
			if y.Time != itemBuilding[x.ItemID][i].Time {
				return fmt.Errorf("items[item_id = %v].building[%v].time が正しくありません : actual %v, expected %v", x.ItemID, i, y.Time, itemBuilding[x.ItemID][i].Time)
			}
			if y.CountBuilt != itemBuilding[x.ItemID][i].CountBuilt {
				return fmt.Errorf("items[item_id = %v].building[%v].count_built が正しくありません : actual %v, expected %v", x.ItemID, i, y.CountBuilt, itemBuilding[x.ItemID][i].CountBuilt)
			}
			if y.Power != itemBuilding[x.ItemID][i].Power {
				return fmt.Errorf("items[item_id = %v].building[%v].power が正しくありません : actual %v, expected %v", x.ItemID, i, y.Power, itemBuilding[x.ItemID][i].Power)
			}
		}
	}

	var statusOnSale = map[int]int64{}
	for _, x := range status.OnSale {
		statusOnSale[x.ItemID] = x.Time
	}
	for itemID := range mItems {
		t1, ok1 := statusOnSale[itemID]
		t2, ok2 := itemOnSale[itemID]
		if ok1 != ok2 {
			if ok1 {
				return fmt.Errorf("on_saleに item_id = %v の要素が存在するのは正しくありません", itemID)
			} else {
				return fmt.Errorf("on_saleに item_id = %v の要素が存在しません", itemID)
			}
		} else if ok1 && t1 != t2 {
			return fmt.Errorf("on_sale[item_id = %v].time が正しくありません : actual %v, expected %v", itemID, t1, t2)
		}
	}

	return nil
}

func calcOnSale(pr *big.Int, ts []int64, xs []*big.Int, vs []*big.Int) int64 {
	if pr.Cmp(xs[len(xs)-1]) > 0 {
		idx := len(ts) - 1
		if vs[idx].Sign() > 0 {
			z := new(big.Int).Add(xs[idx], new(big.Int).Mul(vs[idx], big.NewInt(ts[0]+1000-ts[idx])))
			if pr.Cmp(z) > 0 {
				return -1
			}
			y := new(big.Int).Sub(pr, xs[idx])
			y.Add(y, vs[idx])
			y.Sub(y, big.NewInt(1))
			y.Quo(y, vs[idx])
			if y.IsInt64() {
				t := ts[idx] + y.Int64()
				if t <= ts[0]+1000 {
					return t
				}
			}
		}
		return -1
	} else {
		var r int64 = -1
		for idx := 0; idx < len(ts); idx++ {
			if pr.Cmp(xs[idx]) <= 0 {
				return ts[idx]
			} else {
				if vs[idx].Sign() <= 0 {
					continue
				}
				y := new(big.Int).Sub(pr, xs[idx])
				y.Add(y, vs[idx])
				y.Sub(y, big.NewInt(1))
				y.Quo(y, vs[idx])
				if y.IsInt64() {
					t := ts[idx] + y.Int64()
					if idx+1 < len(ts) && t >= ts[idx+1] {
						continue
					}
					r = t
					break
				}
			}
		}
		return r
	}
}

func validateStatusBench(status *GameStatusLog, addIsuDict, addIsuDictNoRes, addIsuSum, addIsuSumNoRes map[int64]*big.Int, addIsuKeys, addIsuNoResKeys []int64, buyItemDict, buyItemDictNoRes1, buyItemDictNoRes2 map[int]map[int]int64, itemMasterObj *itemMaster) error {
	var (
		// 1ミリ秒に生産できる椅子の単位をミリ椅子とする
		totalMilliIsu1 = big.NewInt(0)
		totalMilliIsu2 = big.NewInt(0)
		totalPower     = big.NewInt(0)

		itemPrice = map[int]*itemMasterElement{}

		itemPower    = map[int]*big.Int{}    // ItemID => Power
		itemOnSale1  = map[int]int64{}       // ItemID => OnSale
		itemOnSale2  = map[int]int64{}       // ItemID => OnSale
		itemBuilt    = map[int]int{}         // ItemID => BuiltCount
		itemBought   = map[int]int{}         // ItemID => CountBought
		itemBuilding = map[int][]Building{}  // ItemID => Buildings
		itemPower0   = map[int]Exponential{} // ItemID => currentTime における Power
		itemBuilt0   = map[int]int{}         // ItemID => currentTime における BuiltCount

		adding   = map[int64]*big.Int{}
		buyingAt = map[int64][]Buying{}
	)

	currentTime := status.Schedule[0].Time

	for itemID := range mItems {
		itemPower[itemID] = big.NewInt(0)
		itemBuilding[itemID] = []Building{}
	}

	for i := len(addIsuKeys) - 1; i >= 0; i-- {
		t := addIsuKeys[i]
		if t <= currentTime {
			v := addIsuSum[t]
			totalMilliIsu1.Add(totalMilliIsu1, v)
			totalMilliIsu2.Add(totalMilliIsu2, v)
			break
		}
	}
	for i := len(addIsuNoResKeys) - 1; i >= 0; i-- {
		t := addIsuNoResKeys[i]
		if t <= currentTime {
			v := addIsuSumNoRes[t]
			totalMilliIsu2.Add(totalMilliIsu2, v)
			break
		}
	}

	for i, a := range status.Adding {
		adding[a.Time] = str2big(a.Isu)
		v1, ok1 := addIsuDict[a.Time]
		v2, ok2 := addIsuDictNoRes[a.Time]
		if !ok1 && !ok2 {
			return fmt.Errorf("adding[%v].time が正しくありません : time = %v のaddIsuが存在しません", i, a.Time)
		} else {
			v := new(big.Int)
			if ok1 {
				v.Add(v, v1)
			}
			if ok2 {
				v.Add(v, v2)
			}
			if adding[a.Time].Cmp(v) > 0 {
				return fmt.Errorf("adding[%v].isu が正しくありません : actual %v, expected %v 以下", i, a.Isu, v.String())
			}
		}
	}

	for _, b := range status.Items {
		a0 := buyItemDict[b.ItemID]
		a1 := buyItemDictNoRes1[b.ItemID]
		a2 := buyItemDictNoRes2[b.ItemID]
		for i := 1; i <= b.CountBought; i++ {
			if _, ok := a0[i-1]; !ok {
				if _, ok := a1[i-1]; !ok {
					return fmt.Errorf("items[item_id = %v].count_bought が正しくありません または count_bought = %v のbuyItemが存在しません : actual %v, expected %v 以下",
						b.ItemID, i-1, b.CountBought, i-1)
				}
				if _, ok := a2[i-1]; !ok {
					return fmt.Errorf("items[item_id = %v].count_bought が正しくありません または count_bought = %v のbuyItemが存在しません : actual %v, expected %v 以下",
						b.ItemID, i-1, b.CountBought, i-1)
				}
			}
		}
	}

	for _, b := range status.Items {
		a0 := buyItemDict[b.ItemID]
		a1 := buyItemDictNoRes1[b.ItemID]
		a2 := buyItemDictNoRes2[b.ItemID]
		itemBuilt[b.ItemID] = b.CountBuilt
		itemBought[b.ItemID] = b.CountBought
		for i := 1; i <= b.CountBuilt; i++ {
			pr := itemMasterObj.getData(b.ItemID, i).v1K
			totalMilliIsu1.Sub(totalMilliIsu1, pr)
			totalMilliIsu2.Sub(totalMilliIsu2, pr)
			power := itemMasterObj.getPower(b.ItemID, i)
			if t0, ok := a0[i-1]; ok {
				if t0 > currentTime {
					return fmt.Errorf("items[item_id = %v].count_built が正しくありません : actual %v, expected %v 以下",
						b.ItemID, b.CountBuilt, i-1)
				}
				x := new(big.Int).Mul(power, big.NewInt(currentTime-t0))
				totalMilliIsu1.Add(totalMilliIsu1, x)
				totalMilliIsu2.Add(totalMilliIsu2, x)
			} else {
				var t1, t2 int64
				t1 = a1[i-1]
				t2 = a2[i-1]
				if t1 > currentTime {
					return fmt.Errorf("items[item_id = %v].count_built が正しくありません : actual %v, expected %v 以下",
						b.ItemID, b.CountBuilt, i-1)
				}
				if t2 > currentTime {
					t2 = currentTime
				}
				totalMilliIsu1.Add(totalMilliIsu1, new(big.Int).Mul(power, big.NewInt(currentTime-t2)))
				totalMilliIsu2.Add(totalMilliIsu2, new(big.Int).Mul(power, big.NewInt(currentTime-t1)))
			}
			totalPower.Add(totalPower, power)
			itemPower[b.ItemID].Add(itemPower[b.ItemID], power)
		}
		bIdx := 0
		for i := b.CountBuilt + 1; i <= b.CountBought; i++ {
			pr := itemMasterObj.getData(b.ItemID, i).v1K
			totalMilliIsu1.Sub(totalMilliIsu1, pr)
			totalMilliIsu2.Sub(totalMilliIsu2, pr)

			if bIdx < len(b.Building) && i > b.Building[bIdx].CountBuilt {
				bIdx += 1
			}
			if bIdx >= len(b.Building) {
				return fmt.Errorf("items[item_id = %v].count_built が正しくありません : actual %v, expected %v 以上",
					b.ItemID, b.CountBuilt, i)
			}

			t := b.Building[bIdx].Time
			if t0, ok := a0[i-1]; ok {
				if t0 != t {
					return fmt.Errorf("items[item_id = %v].building[%v].time が正しくありません : actual %v, expected %v",
						b.ItemID, bIdx, t, t0)
				}
			} else {
				if t > a2[i-1] {
					return fmt.Errorf("items[item_id = %v].building[%v].time が正しくありません : actual %v, expected %v 以下",
						b.ItemID, bIdx, t, a2[i-1])
				}
				tt := a1[i-1]
				if tt < currentTime+1 {
					tt = currentTime + 1
				}
				if t < tt {
					return fmt.Errorf("items[item_id = %v].building[%v].time が正しくありません : actual %v, expected %v 以上",
						b.ItemID, bIdx, t, tt)
				}
			}

			buyingAt[t] = append(buyingAt[t], Buying{
				ItemID:  b.ItemID,
				Ordinal: i,
				Time:    t,
			})
		}
		if t, ok := a0[b.CountBought]; ok {
			if t <= currentTime {
				return fmt.Errorf("items[item_id = %v].count_bought が正しくありません : actual %v, expected %v 以上",
					b.ItemID, b.CountBought, b.CountBought+1)
			}
		}
	}

	for itemID := range mItems {
		itemPower0[itemID] = big2exp(itemPower[itemID])
		itemBuilt0[itemID] = itemBuilt[itemID]
		itemPrice[itemID] = itemMasterObj.getData(itemID, itemBought[itemID]+1)
		if 0 <= totalMilliIsu1.Cmp(itemPrice[itemID].v1K) {
			itemOnSale1[itemID] = 0 // 0 は 時刻 currentTime で購入可能であることを表す
		}
		if 0 <= totalMilliIsu2.Cmp(itemPrice[itemID].v1K) {
			itemOnSale2[itemID] = 0 // 0 は 時刻 currentTime で購入可能であることを表す
		}
	}

	if totalMilliIsu2.Sign() < 0 {
		return fmt.Errorf("時刻 %v で椅子の数が負になります", currentTime)
	}

	var ts []int64
	var x1s []*big.Int
	var x2s []*big.Int
	var vs []*big.Int
	ts = append(ts, currentTime)
	x1s = append(x1s, new(big.Int).Set(totalMilliIsu1))
	x2s = append(x2s, new(big.Int).Set(totalMilliIsu2))
	vs = append(vs, new(big.Int).Set(totalPower))

	// currentTime から 1000 ミリ秒先までシミュレーションする
	timeMap := make(map[int]bool)
	for t := range adding {
		tt := t - currentTime
		if 0 < tt && tt <= 1000 {
			timeMap[int(tt)] = true
		}
	}
	for t := range buyingAt {
		tt := t - currentTime
		if 0 < tt && tt <= 1000 {
			timeMap[int(tt)] = true
		}
	}
	var timeList []int
	for t := range timeMap {
		timeList = append(timeList, t)
	}
	sort.Ints(timeList)
	t1 := currentTime
	for _, ii := range timeList {
		t := currentTime + int64(ii)
		y := new(big.Int).Mul(totalPower, big.NewInt(t-t1))
		totalMilliIsu1.Add(totalMilliIsu1, y)
		totalMilliIsu2.Add(totalMilliIsu2, y)
		t1 = t

		// 時刻 t で発生する adding を計算する
		if v, ok := adding[t]; ok {
			totalMilliIsu1.Add(totalMilliIsu1, new(big.Int).Mul(v, big.NewInt(1000)))
			totalMilliIsu2.Add(totalMilliIsu2, new(big.Int).Mul(v, big.NewInt(1000)))
		}

		// 時刻 t で発生する buying を計算する
		if _, ok := buyingAt[t]; ok {
			updatedID := map[int]bool{}
			for _, b := range buyingAt[t] {
				updatedID[b.ItemID] = true
				power := itemMasterObj.getPower(b.ItemID, b.Ordinal)
				itemPower[b.ItemID].Add(itemPower[b.ItemID], power)
				itemBuilt[b.ItemID]++
				totalPower.Add(totalPower, power)
			}
			for id := range updatedID {
				itemBuilding[id] = append(itemBuilding[id], Building{
					Time:       t,
					CountBuilt: itemBuilt[id],
					Power:      big2exp(itemPower[id]),
				})
			}
		}

		ts = append(ts, t)
		x1s = append(x1s, new(big.Int).Set(totalMilliIsu1))
		x2s = append(x2s, new(big.Int).Set(totalMilliIsu2))
		vs = append(vs, new(big.Int).Set(totalPower))
	}

	var schedule1 []Schedule
	var schedule2 []Exponential
	for i := 0; i < len(ts); i++ {
		var milliIsu1 Exponential
		if x1s[i].Sign() < 0 {
			milliIsu1 = big2exp(big.NewInt(0))
		} else {
			milliIsu1 = big2exp(x1s[i])
		}
		schedule1 = append(schedule1, Schedule{
			Time:       ts[i],
			MilliIsu:   milliIsu1,
			TotalPower: big2exp(vs[i]),
		})
		schedule2 = append(schedule2, big2exp(x2s[i]))
	}

	for itemID := range mItems {
		pr := itemPrice[itemID].v1K
		if _, ok := itemOnSale1[itemID]; !ok {
			if t := calcOnSale(pr, ts, x1s, vs); t >= 0 {
				itemOnSale1[itemID] = t
			}
		}
		if _, ok := itemOnSale2[itemID]; !ok {
			if t := calcOnSale(pr, ts, x2s, vs); t >= 0 {
				itemOnSale2[itemID] = t
			}
		}
	}

	if len(schedule1) != len(schedule2) {
		return fmt.Errorf("something wrong 1")
	}
	if len(status.Schedule) != len(schedule1) {
		return fmt.Errorf("schedule の要素数が正しくありません : actual %v, expected %v", len(status.Schedule), len(schedule1))
	}
	for i, x := range status.Schedule {
		if x.Time != schedule1[i].Time {
			return fmt.Errorf("schedule[%v].time が正しくありません : actual %v, expected %v", i, x.Time, schedule1[i].Time)
		}
		if !(schedule1[i].MilliIsu.LessEq(x.MilliIsu) && x.MilliIsu.LessEq(schedule2[i])) {
			if schedule1[i].MilliIsu.Eq(schedule2[i]) {
				return fmt.Errorf("schedule[%v].milli_isu が正しくありません : actual %v, expected %v", i, x.MilliIsu, schedule1[i].MilliIsu)
			} else {
				return fmt.Errorf("schedule[%v].milli_isu が正しくありません : actual %v, expected %v 以上 %v 以下", i, x.MilliIsu, schedule1[i].MilliIsu, schedule2[i])
			}
		}
		if x.TotalPower != schedule1[i].TotalPower {
			return fmt.Errorf("schedule[%v].total_power が正しくありません : actual %v, expected %v", i, x.TotalPower, schedule1[i].TotalPower)
		}
	}

	for _, x := range status.Items {
		if x.CountBought != itemBought[x.ItemID] {
			return fmt.Errorf("items[item_id = %v].count_bought が正しくありません : actual %v, expected %v", x.ItemID, x.CountBought, itemBought[x.ItemID])
		}
		if x.CountBuilt != itemBuilt0[x.ItemID] {
			return fmt.Errorf("items[item_id = %v].count_built が正しくありません : actual %v, expected %v", x.ItemID, x.CountBuilt, itemBuilt0[x.ItemID])
		}
		if x.NextPrice != itemPrice[x.ItemID].exp {
			return fmt.Errorf("items[item_id = %v].next_price が正しくありません : actual %v, expected %v", x.ItemID, x.NextPrice, itemPrice[x.ItemID].exp)
		}
		if x.Power != itemPower0[x.ItemID] {
			return fmt.Errorf("items[item_id = %v].power が正しくありません : actual %v, expected %v", x.ItemID, x.Power, itemPower0[x.ItemID])
		}
		if len(x.Building) != len(itemBuilding[x.ItemID]) {
			return fmt.Errorf("items[item_id = %v].building の要素数が正しくありません : actual %v, expected %v", x.ItemID, len(x.Building), len(itemBuilding[x.ItemID]))
		}
		for i, y := range x.Building {
			if y.Time != itemBuilding[x.ItemID][i].Time {
				return fmt.Errorf("items[item_id = %v].building[%v].time が正しくありません : actual %v, expected %v", x.ItemID, i, y.Time, itemBuilding[x.ItemID][i].Time)
			}
			if y.CountBuilt != itemBuilding[x.ItemID][i].CountBuilt {
				return fmt.Errorf("items[item_id = %v].building[%v].count_built が正しくありません : actual %v, expected %v", x.ItemID, i, y.CountBuilt, itemBuilding[x.ItemID][i].CountBuilt)
			}
			if y.Power != itemBuilding[x.ItemID][i].Power {
				return fmt.Errorf("items[item_id = %v].building[%v].power が正しくありません : actual %v, expected %v", x.ItemID, i, y.Power, itemBuilding[x.ItemID][i].Power)
			}
		}
	}

	var statusOnSale = map[int]int64{}
	for _, x := range status.OnSale {
		statusOnSale[x.ItemID] = x.Time
	}
	for itemID := range mItems {
		tt, ok0 := statusOnSale[itemID]
		t1, ok1 := itemOnSale2[itemID]
		t2, ok2 := itemOnSale1[itemID]
		if !ok1 {
			if ok2 {
				return fmt.Errorf("something wrong 2")
			}
			if ok0 {
				return fmt.Errorf("on_saleに item_id = %v の要素が存在するのは正しくありません", itemID)
			}
		} else if !ok2 {
			if ok0 && t1 > tt {
				return fmt.Errorf("on_sale[item_id = %v].time が正しくありません : actual %v, expected %v 以上 あるいは 要素なし", itemID, tt, t1)
			}
		} else {
			if !ok0 {
				return fmt.Errorf("on_saleに item_id = %v の要素が存在しません", itemID)
			}
			if !(t1 <= tt && tt <= t2) {
				return fmt.Errorf("on_sale[item_id = %v].time が正しくありません : actual %v, expected %v 以上 %v 以下", itemID, tt, t1, t2)
			}
		}
	}

	return nil
}

func readGameLogGob(path string) (*gameLogDump, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	g, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer g.Close()

	dst := new(gameLogDump)
	err = gob.NewDecoder(g).Decode(dst)
	if err != nil {
		return nil, err
	}

	return dst, nil
}

func dumpGameLogGob(name string, data gameLogDump) error {
	f, err := ioutil.TempFile("", fmt.Sprintf("isu7f-gamelog-%v", name))
	if err != nil {
		return err
	}
	defer f.Close()

	g := gzip.NewWriter(f)
	defer g.Close()

	err = gob.NewEncoder(g).Encode(data)
	if err != nil {
		log.Println("Failed to save gamelog dump:", f.Name())
	} else {
		log.Println("save gamelog dump to:", f.Name())
	}
	return err
}

func ValidateGameLog(ctx context.Context, room string, isPreTest bool) error {
	g := getGameLogger(room)

	g.mtx.Lock()
	var (
		status   = g.status
		request  = g.request
		response = g.response
	)
	g.mtx.Unlock()

	log.Println("ValidateGameLog", room, len(status), len(request), len(response))
	err := validateGameLog(ctx, isPreTest, status, request, response)
	log.Println("ValidateGameLog end", room)

	if (err != nil && saveGameLogDump == 1) || saveGameLogDump == 2 {
		dumpGameLogGob(room, gameLogDump{
			Room:        room,
			IsPreTest:   isPreTest,
			StatusLog:   status,
			RequestLog:  request,
			ResponseLog: response,
		})
	}
	if err != nil {
		return err
	}

	return nil
}

func ValidateGameLogDump(path string) error {
	g, err := readGameLogGob(path)
	if err != nil {
		return err
	}

	var (
		status   = g.StatusLog
		request  = g.RequestLog
		response = g.ResponseLog
	)

	log.Println("ValidateGameLogDump", g.Room, len(status), len(request), len(response))
	err = validateGameLog(context.Background(), g.IsPreTest, status, request, response)
	if err != nil {
		return err
	}

	log.Println("ValidateGameLogDump end", g.Room)
	return nil
}

func validateGameLog(ctx context.Context, isPreTest bool, status []*GameStatusLog, request []*GameRequestLog, response []*GameResponseLog) error {
	statusDict := make(map[int][]*GameStatusLog)
	for _, st := range status {
		if len(st.Schedule) <= 0 {
			return fmt.Errorf("scheduleが空です")
		}
		for i := 0; i < len(st.Schedule)-1; i++ {
			if st.Schedule[i].Time >= st.Schedule[i+1].Time {
				return fmt.Errorf("schedule の順序が正しくありません : schedule[%v].time = %v, schedule[%v].time = %v",
					i, st.Schedule[i].Time, i+1, st.Schedule[i+1].Time)
			}
		}
		if len(st.Items) != len(mItems) {
			return fmt.Errorf("items の要素数が正しくありません : actual %v, expected %v", len(st.Items), len(mItems))
		}
		a := make(map[int]bool)
		for _, x := range st.Items {
			if _, ok := a[x.ItemID]; ok {
				return fmt.Errorf("items に同一のitem_idの要素が存在します : item_id = %v", x.ItemID)
			}
			if _, ok := mItems[x.ItemID]; !ok {
				return fmt.Errorf("items に正しくないitem_idの要素が存在します : item_id = %v", x.ItemID)
			}
			a[x.ItemID] = true
		}

		statusDict[st.ClientID] = append(statusDict[st.ClientID], st)
	}
	for _, a := range statusDict {
		sort.Slice(a, func(i, j int) bool {
			return a[i].ClientTime.Before(a[j].ClientTime)
		})
	}

	responseDict := make(map[int]*GameResponseLog)
	for _, x := range response {
		if _, ok := responseDict[x.RequestID]; ok {
			return fmt.Errorf("request_id = %v に対するreponseが複数あります", x.RequestID)
		}
		responseDict[x.RequestID] = x
	}

	addIsuDict := make(map[int64]*big.Int)
	addIsuDictNoRes := make(map[int64]*big.Int)
	buyItemDict := make(map[int]map[int]int64)
	buyItemDictNoRes1 := make(map[int]map[int]int64)
	buyItemDictNoRes2 := make(map[int]map[int]int64)
	for itemID := range mItems {
		buyItemDict[itemID] = make(map[int]int64)
		buyItemDictNoRes1[itemID] = make(map[int]int64)
		buyItemDictNoRes2[itemID] = make(map[int]int64)
	}
	for _, req := range request {
		res, ok := responseDict[req.RequestID]
		if !ok {
			if isPreTest {
				return fmt.Errorf("request_id = %v に対するresponseが存在しません", req.RequestID)
			} else {
				if req.Action == "addIsu" {
					if _, ok := addIsuDictNoRes[req.Time]; !ok {
						addIsuDictNoRes[req.Time] = new(big.Int)
					}
					addIsuDictNoRes[req.Time].Add(addIsuDictNoRes[req.Time], str2big(req.Isu))
				} else if req.Action == "buyItem" {
					if t, ok := buyItemDictNoRes1[req.ItemID][req.CountBought]; !ok || req.Time < t {
						buyItemDictNoRes1[req.ItemID][req.CountBought] = req.Time
					}
					if t, ok := buyItemDictNoRes2[req.ItemID][req.CountBought]; !ok || req.Time > t {
						buyItemDictNoRes2[req.ItemID][req.CountBought] = req.Time
					}
				} else {
					return fmt.Errorf("something wrong 3")
				}
			}
		} else {
			if req.ClientID != res.ClientID {
				return fmt.Errorf("request_id = %v に対するrequestとresponseでclient idが一致していません", req.RequestID)
			}
			if res.IsSuccess {
				status := getLatestStatus(statusDict[req.ClientID], req.ClientTime, res.ClientTime)
				if status == nil {
					return fmt.Errorf("request_id = %v に対するstatusを受信していません", req.RequestID)
				}

				if req.Action == "addIsu" {
					if err := validateAddIsu(req.RequestID, req.Time, req.Isu, status); err != nil {
						return err
					}
					if _, ok := addIsuDict[req.Time]; !ok {
						addIsuDict[req.Time] = new(big.Int)
					}
					addIsuDict[req.Time].Add(addIsuDict[req.Time], str2big(req.Isu))
				} else if req.Action == "buyItem" {
					if err := validateBuyItem(req.RequestID, req.ItemID, req.CountBought, status); err != nil {
						return err
					}
					if _, ok := buyItemDict[req.ItemID][req.CountBought]; ok {
						return fmt.Errorf("buyItem(item_id = %v, count_bought = %v) に対して複数回 is_success = true が存在します : request_id = %v",
							req.ItemID, req.CountBought, req.RequestID)
					}
					buyItemDict[req.ItemID][req.CountBought] = req.Time
				} else {
					return fmt.Errorf("something wrong 4")
				}
			}
		}
	}

	itemMasterObj := newItemMaster()
	if isPreTest {
		for itemID, a := range buyItemDict {
			for i := 0; i < len(a); i++ {
				if _, ok := a[i]; !ok {
					return fmt.Errorf("buyItem(item_id = %v, count_bought = %v) に対するresponseを受信していません", itemID, i)
				}
			}
			for i := 1; i < len(a); i++ {
				if a[i-1] > a[i] {
					return fmt.Errorf("item_id = %v に対するbuyItemの順序が正しくありません : count_bought = %v の時刻 %v, count_bought = %v の時刻 %v",
						itemID, i-1, a[i-1], i, a[i])
				}
			}
		}

		for _, a := range statusDict {
			for _, x := range a {
				if err := validateStatus(x, addIsuDict, buyItemDict, itemMasterObj); err != nil {
					return fmt.Errorf("time = %v の status において %v", x.Time, err)
				}
				if ctx.Err() != nil {
					return nil
				}
			}
		}
	} else {
		var x *big.Int
		big1000 := big.NewInt(1000)

		var addIsuKeys []int64
		for t := range addIsuDict {
			addIsuKeys = append(addIsuKeys, t)
		}
		sort.Slice(addIsuKeys, func(i, j int) bool {
			return addIsuKeys[i] < addIsuKeys[j]
		})
		addIsuSum := make(map[int64]*big.Int)
		x = big.NewInt(0)
		for _, t := range addIsuKeys {
			x = new(big.Int).Add(x, new(big.Int).Mul(addIsuDict[t], big1000))
			addIsuSum[t] = x
		}

		var addIsuNoResKeys []int64
		for t := range addIsuDictNoRes {
			addIsuNoResKeys = append(addIsuNoResKeys, t)
		}
		sort.Slice(addIsuNoResKeys, func(i, j int) bool {
			return addIsuNoResKeys[i] < addIsuNoResKeys[j]
		})
		addIsuSumNoRes := make(map[int64]*big.Int)
		x = big.NewInt(0)
		for _, t := range addIsuNoResKeys {
			x = new(big.Int).Add(x, new(big.Int).Mul(addIsuDictNoRes[t], big1000))
			addIsuSumNoRes[t] = x
		}

		for _, a := range statusDict {
			for _, x := range a {
				if err := validateStatusBench(
					x, addIsuDict, addIsuDictNoRes, addIsuSum, addIsuSumNoRes, addIsuKeys, addIsuNoResKeys,
					buyItemDict, buyItemDictNoRes1, buyItemDictNoRes2, itemMasterObj); err != nil {
					return fmt.Errorf("time = %v の status において %v", x.Time, err)
				}
				if ctx.Err() != nil {
					return nil
				}
			}
		}
	}

	return nil
}
