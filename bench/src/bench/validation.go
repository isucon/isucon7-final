package main

import "fmt"

func validateGameResponseFormat(res *GameResponse) error {
	if res.RequestID <= 0 {
		return fmt.Errorf("request_id の値が正しくありません")
	}
	return nil
}

func validateGameStatusFormat(st *GameStatus) error {
	if len(st.Schedule) == 0 {
		return fmt.Errorf("schedule 配列が空です")
	}

	if st.Time <= 0 {
		return fmt.Errorf("time の値が正しくありません")
	}

	cTime := st.Schedule[0].Time

	for i := 0; i < len(st.Schedule)-1; i++ {
		a := st.Schedule[i]
		b := st.Schedule[i+1]

		if a.Time > b.Time {
			return fmt.Errorf("schedule の時刻が昇順ではありません")
		}
		if a.Time == b.Time {
			return fmt.Errorf("schedule に同一の時刻が含まれています")
		}
		if b.MilliIsu.Less(a.MilliIsu) {
			return fmt.Errorf("schedule で milli_isu が減少しています")
		}
		if b.TotalPower.Less(a.TotalPower) {
			return fmt.Errorf("schedule で total_power が減少しています")
		}
	}

	if len(st.Items) != len(mItems) {
		return fmt.Errorf("itemsの要素数が正しくありません")
	}
	itemChecked := map[int]bool{}
	for _, item := range st.Items {
		if _, ok := mItems[item.ItemID]; !ok {
			return fmt.Errorf("items に含まれている item_id が正しくありません")
		}
		if itemChecked[item.ItemID] {
			return fmt.Errorf("items に含まれている item_id が正しくありません")
		}
		itemChecked[item.ItemID] = true

		if item.CountBought < item.CountBuilt {
			return fmt.Errorf("items に含まれている count_built の数が count_bought の数を上回っています")
		}
		if item.CountBuilt == 0 {
			if !(item.Power.Exponent == 0 && item.Power.Mantissa == 0) {
				return fmt.Errorf("items に含まれている power の値が正しくありません")
			}
		}
	}

	for _, a := range st.Adding {
		if a.Isu == "" {
			return fmt.Errorf("adding の isu が空です")
		}
		if a.Time < cTime {
			return fmt.Errorf("adding に schedule[0].time よりも過去の情報が含まれています")
		}
		if a.Time == cTime {
			return fmt.Errorf("adding に schedule[0].time 時点の情報が含まれています")
		}
	}

	for _, o := range st.OnSale {
		if _, ok := mItems[o.ItemID]; !ok {
			return fmt.Errorf("on_sale に含まれている item_id が正しくありません")
		}
		if 0 < o.Time && o.Time < cTime {
			return fmt.Errorf("on_sale に schedule[0].time よりも過去の情報が含まれています")
		}
		if 0 < o.Time && o.Time == cTime {
			return fmt.Errorf("on_sale に schedule[0].time 時点の情報が含まれています")
		}
	}

	return nil
}
