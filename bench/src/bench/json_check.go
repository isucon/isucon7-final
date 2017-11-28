package main

import jp "github.com/buger/jsonparser"

func checkSameAdding(b []byte, adding []Adding) bool {
	idx := 0
	res := true

	if _, _, _, err := jp.Get(b, "adding"); err != nil {
		return false
	}
	jp.ArrayEach(b, func(value []byte, _ jp.ValueType, _ int, _ error) {
		if idx == len(adding) {
			res = false
			return
		}
		a := adding[idx]
		idx++

		if time, e := jp.GetInt(value, "time"); e != nil || a.Time != time {
			res = false
			return
		}
		if isu, e := jp.GetUnsafeString(value, "isu"); e != nil || a.Isu != isu {
			res = false
			return
		}
	}, "adding")

	if !res {
		return false
	}

	if idx != len(adding) {
		return false
	}

	return true
}

func checkSameSchedule(b []byte, schedule []Schedule) bool {
	idx := 0
	res := true

	if _, _, _, err := jp.Get(b, "schedule"); err != nil {
		return false
	}
	jp.ArrayEach(b, func(value []byte, _ jp.ValueType, _ int, _ error) {
		if idx == len(schedule) {
			res = false
			return
		}
		s := schedule[idx]
		idx++

		if time, e := jp.GetInt(value, "time"); e != nil || s.Time != time {
			res = false
			return
		}
		if milli_isu0, e := jp.GetInt(value, "milli_isu", "[0]"); e != nil || s.MilliIsu.Mantissa != milli_isu0 {
			res = false
			return
		}
		if milli_isu1, e := jp.GetInt(value, "milli_isu", "[1]"); e != nil || s.MilliIsu.Exponent != milli_isu1 {
			res = false
			return
		}
		if total_power0, e := jp.GetInt(value, "total_power", "[0]"); e != nil || s.TotalPower.Mantissa != total_power0 {
			res = false
			return
		}
		if total_power1, e := jp.GetInt(value, "total_power", "[1]"); e != nil || s.TotalPower.Exponent != total_power1 {
			res = false
			return
		}
	}, "schedule")

	if !res {
		return false
	}

	if idx != len(schedule) {
		return false
	}

	return true
}

func checkSameItems(b []byte, items []Item) bool {
	idx := 0
	res := true

	if _, _, _, err := jp.Get(b, "items"); err != nil {
		return false
	}
	jp.ArrayEach(b, func(value []byte, _ jp.ValueType, _ int, _ error) {
		if idx == len(items) {
			res = false
			return
		}

		i := items[idx]
		idx++

		if item_id, e := jp.GetInt(value, "item_id"); e != nil || i.ItemID != int(item_id) {
			res = false
			return
		}

		if count_bought, e := jp.GetInt(value, "count_bought"); e != nil || i.CountBought != int(count_bought) {
			res = false
			return
		}

		if count_built, e := jp.GetInt(value, "count_built"); e != nil || i.CountBuilt != int(count_built) {
			res = false
			return
		}

		if next_price0, e := jp.GetInt(value, "next_price", "[0]"); e != nil || i.NextPrice.Mantissa != next_price0 {
			res = false
			return
		}

		if next_price1, e := jp.GetInt(value, "next_price", "[1]"); e != nil || i.NextPrice.Exponent != next_price1 {
			res = false
			return
		}

		if power0, e := jp.GetInt(value, "power", "[0]"); e != nil || i.Power.Mantissa != power0 {
			res = false
			return
		}

		if power1, e := jp.GetInt(value, "power", "[1]"); e != nil || i.Power.Exponent != power1 {
			res = false
			return
		}

		if _, _, _, e := jp.Get(value, "building"); e != nil {
			res = false
			return
		}

		bidx := 0
		jp.ArrayEach(value, func(value []byte, _ jp.ValueType, _ int, _ error) {
			if bidx == len(i.Building) {
				res = false
				return
			}

			b := i.Building[bidx]
			bidx++

			if time, e := jp.GetInt(value, "time"); e != nil || b.Time != time {
				res = false
				return
			}

			if count_built, e := jp.GetInt(value, "count_built"); e != nil || b.CountBuilt != int(count_built) {
				res = false
				return
			}

			if power0, e := jp.GetInt(value, "power", "[0]"); e != nil || b.Power.Mantissa != power0 {
				res = false
				return
			}

			if power1, e := jp.GetInt(value, "power", "[1]"); e != nil || b.Power.Exponent != power1 {
				res = false
				return
			}
		}, "building")
		if bidx != len(i.Building) {
			res = false
			return
		}
		if !res {
			return
		}
	}, "items")

	if !res {
		return false
	}

	if idx != len(items) {
		return false
	}

	return true

}

func checkSameOnSale(b []byte, onSale []OnSale) bool {
	idx := 0
	res := true

	if _, _, _, err := jp.Get(b, "on_sale"); err != nil {
		return false
	}
	jp.ArrayEach(b, func(value []byte, _ jp.ValueType, _ int, _ error) {
		if idx == len(onSale) {
			res = false
			return
		}
		o := onSale[idx]
		idx++

		if item_id, e := jp.GetInt(value, "item_id"); e != nil || o.ItemID != int(item_id) {
			res = false
			return
		}
		if time, e := jp.GetInt(value, "time"); e != nil || o.Time != time {
			res = false
			return
		}
	}, "on_sale")

	if !res {
		return false
	}

	if idx != len(onSale) {
		return false
	}

	return true
}

func checkSameJson(b []byte, st *GameStatus) bool {
	// st.Time はチェックしないことに注意

	if !checkSameAdding(b, st.Adding) {
		return false
	}

	if !checkSameSchedule(b, st.Schedule) {
		return false
	}

	if !checkSameItems(b, st.Items) {
		return false
	}

	if !checkSameOnSale(b, st.OnSale) {
		return false
	}

	return true
}
