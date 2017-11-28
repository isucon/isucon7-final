package main

import (
	"fmt"
	"hash"

	jp "github.com/buger/jsonparser"
)

func jpGetWrite(b []byte, h hash.Hash64, key string) error {
	x, _, _, err := jp.Get(b, key)
	if err != nil {
		return err
	}

	_, err = h.Write(x)
	if err != nil {
		return err
	}

	return nil
}

func hashAddingJson(b []byte, h hash.Hash64) (uint64, error) {
	var err error
	h.Reset()

	_, errEach := jp.ArrayEach(b, func(b []byte, _ jp.ValueType, _ int, _ error) {
		if err != nil {
			return
		}

		for _, key := range []string{
			"time",
			"isu",
		} {
			err = jpGetWrite(b, h, key)
			if err != nil {
				err = fmt.Errorf("%v adding."+key, err)
				return
			}
		}
	}, "adding")
	if errEach != nil {
		return 0, fmt.Errorf("%v adding", errEach)
	}
	if err != nil {
		return 0, err
	}

	return h.Sum64(), nil
}

func hashScheduleJson(b []byte, h hash.Hash64) (uint64, error) {
	var err error
	h.Reset()

	_, errEach := jp.ArrayEach(b, func(b []byte, _ jp.ValueType, _ int, _ error) {
		if err != nil {
			return
		}

		for _, key := range []string{
			"time",
			"milli_isu",
			"total_power",
		} {
			err = jpGetWrite(b, h, key)
			if err != nil {
				err = fmt.Errorf("%v schedule."+key, err)
				return
			}
		}
	}, "schedule")
	if errEach != nil {
		return 0, fmt.Errorf("%v schedule", errEach)
	}
	if err != nil {
		return 0, err
	}

	return h.Sum64(), nil
}

func hashItemsJson(b []byte, h hash.Hash64) (uint64, error) {
	var err error
	h.Reset()

	_, errEach := jp.ArrayEach(b, func(b []byte, _ jp.ValueType, _ int, _ error) {
		if err != nil {
			return
		}

		for _, key := range []string{
			"item_id",
			"count_bought",
			"count_built",
			"next_price",
			"power",
		} {
			err = jpGetWrite(b, h, key)
			if err != nil {
				err = fmt.Errorf("%v items."+key, err)
				return
			}
		}

		_, errEach := jp.ArrayEach(b, func(b []byte, _ jp.ValueType, _ int, _ error) {
			if err != nil {
				return
			}

			for _, key := range []string{
				"time",
				"count_built",
				"power",
			} {
				err = jpGetWrite(b, h, key)
				if err != nil {
					err = fmt.Errorf("%v items.building."+key, err)
					return
				}
			}
		}, "building")
		if errEach != nil {
			err = fmt.Errorf("%v items.building", errEach)
			return
		}
	}, "items")
	if errEach != nil {
		return 0, fmt.Errorf("%v items", errEach)
	}
	if err != nil {
		return 0, err
	}

	return h.Sum64(), nil
}

func hashOnSaleJson(b []byte, h hash.Hash64) (uint64, error) {
	var err error
	h.Reset()

	_, errEach := jp.ArrayEach(b, func(b []byte, _ jp.ValueType, _ int, _ error) {
		if err != nil {
			return
		}

		for _, key := range []string{
			"item_id",
			"time",
		} {
			err = jpGetWrite(b, h, key)
			if err != nil {
				err = fmt.Errorf("%v on_sale."+key, err)
				return
			}
		}
	}, "on_sale")
	if errEach != nil {
		return 0, fmt.Errorf("%v on_sale", errEach)
	}
	if err != nil {
		return 0, err
	}

	return h.Sum64(), nil
}
