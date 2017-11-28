package main

import (
	"hash"

	jp "github.com/buger/jsonparser"
)

func decodeReponseJson(b []byte, hasher hash.Hash64) (interface{}, error) {
	// request_id があれば GameResponse それ以外は GameStatus
	request_id, err := jp.GetInt(b, "request_id")
	if err == nil {
		is_success, err := jp.GetBoolean(b, "is_success")
		if err != nil {
			return nil, err
		}
		return &GameResponse{
			RequestID: int(request_id),
			IsSuccess: is_success,
		}, nil
	}

	time, err := jp.GetInt(b, "time")
	if err != nil {
		return nil, err
	}

	hasher.Reset()
	hasher.Write(b)
	binHash := hasher.Sum64()

	if gs, ok := retrieveBinCache(b, binHash); ok {
		if gs.Time == time {
			return gs, nil
		}
	}

	gs := &GameStatus{
		Time: time,
	}

	// 各要素は, キャッシュがあれば使用し, なければデコードしてキャッシュする.

	if h, err := hashAddingJson(b, hasher); err != nil {
		return nil, err
	} else if v, ok := retrieveAddingCache(b, h); ok {
		gs.Adding = v
	} else if v, err := decodeAddingJson(b); err != nil {
		return nil, err
	} else {
		gs.Adding = v
		storeAddingCache(h, v)
	}

	if h, err := hashScheduleJson(b, hasher); err != nil {
		return nil, err
	} else if v, ok := retrieveScheduleCache(b, h); ok {
		gs.Schedule = v
	} else if v, err := decodeScheduleJson(b); err != nil {
		return nil, err
	} else {
		gs.Schedule = v
		storeScheduleCache(h, v)
	}

	if h, err := hashItemsJson(b, hasher); err != nil {
		return nil, err
	} else if v, ok := retrieveItemsCache(b, h); ok {
		gs.Items = v
	} else if v, err := decodeItemsJson(b); err != nil {
		return nil, err
	} else {
		gs.Items = v
		storeItemsCache(h, v)
	}

	if h, err := hashOnSaleJson(b, hasher); err != nil {
		return nil, err
	} else if v, ok := retrieveOnSaleCache(b, h); ok {
		gs.OnSale = v
	} else if v, err := decodeOnSaleJson(b); err != nil {
		return nil, err
	} else {
		gs.OnSale = v
		storeOnSaleCache(h, v)
	}

	storeBinCache(binHash, gs)
	return gs, nil
}

func decodeAddingJson(b []byte) ([]Adding, error) {
	if _, _, _, err := jp.Get(b, "adding"); err != nil {
		return nil, err
	}
	adding := []Adding{}
	var err error
	jp.ArrayEach(b, func(value []byte, _ jp.ValueType, _ int, _ error) {
		time, e := jp.GetInt(value, "time")
		if e != nil {
			err = e
			return
		}
		isu, e := jp.GetString(value, "isu")
		if e != nil {
			err = e
			return
		}
		adding = append(adding, Adding{Time: time, Isu: isu})
	}, "adding")
	if err != nil {
		return nil, err
	}
	return adding, err
}

func decodeScheduleJson(b []byte) ([]Schedule, error) {
	if _, _, _, err := jp.Get(b, "schedule"); err != nil {
		return nil, err
	}
	schedule := []Schedule{}
	var err error
	jp.ArrayEach(b, func(value []byte, _ jp.ValueType, _ int, _ error) {
		time, e := jp.GetInt(value, "time")
		if e != nil {
			err = e
			return
		}
		milli_isu0, e := jp.GetInt(value, "milli_isu", "[0]")
		if e != nil {
			err = e
			return
		}
		milli_isu1, e := jp.GetInt(value, "milli_isu", "[1]")
		if e != nil {
			err = e
			return
		}
		total_power0, e := jp.GetInt(value, "total_power", "[0]")
		if e != nil {
			err = e
			return
		}
		total_power1, e := jp.GetInt(value, "total_power", "[1]")
		if e != nil {
			err = e
			return
		}
		schedule = append(schedule, Schedule{
			Time:       time,
			MilliIsu:   Exponential{milli_isu0, milli_isu1},
			TotalPower: Exponential{total_power0, total_power1},
		})
	}, "schedule")
	if err != nil {
		return nil, err
	}
	return schedule, nil
}

func decodeItemsJson(b []byte) ([]Item, error) {
	if _, _, _, err := jp.Get(b, "items"); err != nil {
		return nil, err
	}
	items := []Item{}
	var err error
	jp.ArrayEach(b, func(value []byte, _ jp.ValueType, _ int, _ error) {
		item_id, e := jp.GetInt(value, "item_id")
		if e != nil {
			err = e
			return
		}
		count_bought, e := jp.GetInt(value, "count_bought")
		if e != nil {
			err = e
			return
		}
		count_built, e := jp.GetInt(value, "count_built")
		if e != nil {
			err = e
			return
		}
		next_price0, e := jp.GetInt(value, "next_price", "[0]")
		if e != nil {
			err = e
			return
		}
		next_price1, e := jp.GetInt(value, "next_price", "[1]")
		if e != nil {
			err = e
			return
		}
		power0, e := jp.GetInt(value, "power", "[0]")
		if e != nil {
			err = e
			return
		}
		power1, e := jp.GetInt(value, "power", "[1]")
		if e != nil {
			err = e
			return
		}

		building := []Building{}
		if _, _, _, e := jp.Get(value, "building"); e != nil {
			err = e
			return
		}
		jp.ArrayEach(value, func(value []byte, _ jp.ValueType, _ int, _ error) {
			time, e := jp.GetInt(value, "time")
			if e != nil {
				err = e
				return
			}
			count_built, e := jp.GetInt(value, "count_built")
			if e != nil {
				err = e
				return
			}
			power0, e := jp.GetInt(value, "power", "[0]")
			if e != nil {
				err = e
				return
			}
			power1, e := jp.GetInt(value, "power", "[1]")
			if e != nil {
				err = e
				return
			}
			building = append(building, Building{
				Time:       time,
				CountBuilt: int(count_built),
				Power:      Exponential{power0, power1},
			})
		}, "building")
		if err != nil {
			return
		}

		items = append(items, Item{
			ItemID:      int(item_id),
			CountBought: int(count_bought),
			CountBuilt:  int(count_built),
			NextPrice:   Exponential{next_price0, next_price1},
			Power:       Exponential{power0, power1},
			Building:    building,
		})
	}, "items")
	if err != nil {
		return nil, err
	}

	return items, nil
}

func decodeOnSaleJson(b []byte) ([]OnSale, error) {
	if _, _, _, err := jp.Get(b, "on_sale"); err != nil {
		return nil, err
	}
	onsale := []OnSale{}
	var err error
	jp.ArrayEach(b, func(value []byte, _ jp.ValueType, _ int, _ error) {
		item_id, e := jp.GetInt(value, "item_id")
		if e != nil {
			err = e
			return
		}
		time, e := jp.GetInt(value, "time")
		if e != nil {
			err = e
			return
		}
		onsale = append(onsale, OnSale{
			ItemID: int(item_id),
			Time:   time,
		})
	}, "on_sale")
	if err != nil {
		return nil, err
	}
	return onsale, nil
}
