package main

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusEmpty(t *testing.T) {
	assert := assert.New(t)

	mItems := map[int]mItem{}
	addings := []Adding{}
	buyings := []Buying{}

	s, err := calcStatus(0, mItems, addings, buyings)

	assert.Nil(err)
	assert.Empty(s.Adding)
	assert.Len(s.Schedule, 1)
	assert.Empty(s.OnSale)

	assert.Equal(int64(0), s.Schedule[0].Time)
	assert.Equal(Exponential{0, 0}, s.Schedule[0].MilliIsu)
	assert.Equal(Exponential{0, 0}, s.Schedule[0].TotalPower)
}

// 椅子が増える
func TestStatusAdd(t *testing.T) {
	assert := assert.New(t)

	mItems := map[int]mItem{}
	addings := []Adding{
		Adding{Time: 100, Isu: "1"},
		Adding{Time: 200, Isu: "2"},
		Adding{Time: 300, Isu: "1234567890123456789"},
	}
	buyings := []Buying{}

	s, err := calcStatus(0, mItems, addings, buyings)
	assert.Nil(err)
	assert.Len(s.Adding, 3)
	assert.Len(s.Schedule, 4)

	assert.Equal(int64(0), s.Schedule[0].Time)
	assert.Equal(Exponential{0, 0}, s.Schedule[0].MilliIsu)
	assert.Equal(Exponential{0, 0}, s.Schedule[0].TotalPower)

	assert.Equal(int64(100), s.Schedule[1].Time)
	assert.Equal(Exponential{1000, 0}, s.Schedule[1].MilliIsu)
	assert.Equal(Exponential{0, 0}, s.Schedule[1].TotalPower)

	assert.Equal(int64(200), s.Schedule[2].Time)
	assert.Equal(Exponential{3000, 0}, s.Schedule[2].MilliIsu)
	assert.Equal(Exponential{0, 0}, s.Schedule[2].TotalPower)

	assert.Equal(int64(300), s.Schedule[3].Time)
	assert.Equal(Exponential{123456789012345, 7}, s.Schedule[3].MilliIsu)
	assert.Equal(Exponential{0, 0}, s.Schedule[3].TotalPower)

	s, err = calcStatus(500, mItems, addings, buyings)
	assert.Nil(err)
	assert.Len(s.Adding, 0)
	assert.Len(s.Schedule, 1)

	assert.Equal(int64(500), s.Schedule[0].Time)
	assert.Equal(Exponential{123456789012345, 7}, s.Schedule[0].MilliIsu)
	assert.Equal(Exponential{0, 0}, s.Schedule[0].TotalPower)
}

// 試しに１個買う
func TestStatusBuySingle(t *testing.T) {
	assert := assert.New(t)
	x := mItem{
		ItemID: 1,
		Power1: 0, Power2: 1, Power3: 0, Power4: 10,
		Price1: 0, Price2: 1, Price3: 0, Price4: 10,
	}
	mItems := map[int]mItem{1: x}
	initialIsu := "10"
	addings := []Adding{
		Adding{Time: 0, Isu: initialIsu},
	}
	buyings := []Buying{
		Buying{ItemID: 1, Ordinal: 1, Time: 100},
	}
	s, err := calcStatus(0, mItems, addings, buyings)
	assert.Nil(err)
	assert.Len(s.Adding, 0)
	assert.Len(s.Schedule, 2)
	assert.Len(s.Items, 1)

	assert.Equal(int64(0), s.Schedule[0].Time)
	assert.Equal(Exponential{0, 0}, s.Schedule[0].MilliIsu)
	assert.Equal(Exponential{0, 0}, s.Schedule[0].TotalPower)

	assert.Equal(int64(100), s.Schedule[1].Time)
	assert.Equal(Exponential{0, 0}, s.Schedule[1].MilliIsu)
	assert.Equal(Exponential{10, 0}, s.Schedule[1].TotalPower)
}

// 購入時間を見ます
func TestOnSale(t *testing.T) {
	assert := assert.New(t)
	x := mItem{
		ItemID: 1,
		Power1: 0, Power2: 1, Power3: 0, Power4: 1, // power: (0x+1)*1^(0x+1)
		Price1: 0, Price2: 1, Price3: 0, Price4: 1, // price: (0x+1)*1^(0x+1)
	}
	mItems := map[int]mItem{1: x}
	addings := []Adding{Adding{Time: 0, Isu: "1"}}
	buyings := []Buying{Buying{ItemID: 1, Ordinal: 1, Time: 0}}

	s, err := calcStatus(1, mItems, addings, buyings)
	assert.Nil(err)
	assert.Len(s.Adding, 0)
	assert.Len(s.Schedule, 1)
	assert.Len(s.OnSale, 1)
	assert.Len(s.Items, 1)

	assert.Equal(int64(1), s.Schedule[0].Time)
	assert.Equal(OnSale{ItemID: 1, Time: 1000}, s.OnSale[0])

	assert.Equal(s.Items[0].CountBought, 1)
	assert.Equal(s.Items[0].Power, Exponential{1, 0})
	assert.Equal(s.Items[0].CountBuilt, 1)
	assert.Equal(s.Items[0].NextPrice, Exponential{1, 0})
}

func TestStatusBuy(t *testing.T) {
	assert := assert.New(t)

	x := mItem{
		ItemID: 1,
		Power1: 1, Power2: 1, Power3: 3, Power4: 2,
		Price1: 1, Price2: 1, Price3: 7, Price4: 6,
	}
	y := mItem{
		ItemID: 2,
		Power1: 1, Power2: 1, Power3: 7, Power4: 6,
		Price1: 1, Price2: 1, Price3: 3, Price4: 2,
	}
	mItems := map[int]mItem{1: x, 2: y}
	initialIsu := "10000000"
	addings := []Adding{
		Adding{Time: 0, Isu: initialIsu},
	}
	buyings := []Buying{
		Buying{ItemID: 1, Ordinal: 1, Time: 100},
		Buying{ItemID: 1, Ordinal: 2, Time: 200},
		Buying{ItemID: 2, Ordinal: 1, Time: 300},
		Buying{ItemID: 2, Ordinal: 2, Time: 2001},
	}

	s, err := calcStatus(0, mItems, addings, buyings)
	assert.Nil(err)
	assert.Len(s.Adding, 0)
	assert.Len(s.Schedule, 4)
	assert.Len(s.OnSale, 2)
	assert.Len(s.Items, 2)

	totalPower := big.NewInt(0)
	milliIsu := new(big.Int).Mul(str2big(initialIsu), big.NewInt(1000))
	milliIsu.Sub(milliIsu, new(big.Int).Mul(x.GetPrice(1), big.NewInt(1000)))
	milliIsu.Sub(milliIsu, new(big.Int).Mul(x.GetPrice(2), big.NewInt(1000)))
	milliIsu.Sub(milliIsu, new(big.Int).Mul(y.GetPrice(1), big.NewInt(1000)))
	milliIsu.Sub(milliIsu, new(big.Int).Mul(y.GetPrice(2), big.NewInt(1000)))

	// 0sec
	assert.Equal(int64(0), s.Schedule[0].Time)
	assert.Equal(big2exp(milliIsu), s.Schedule[0].MilliIsu)
	assert.Equal(big2exp(totalPower), s.Schedule[0].TotalPower)

	// 0.1sec
	totalPower.Add(totalPower, x.GetPower(1))
	assert.Equal(int64(100), s.Schedule[1].Time)
	assert.Equal(big2exp(milliIsu), s.Schedule[1].MilliIsu)
	assert.Equal(big2exp(totalPower), s.Schedule[1].TotalPower)

	// 0.2sec
	milliIsu.Add(milliIsu, new(big.Int).Mul(totalPower, big.NewInt(100)))
	totalPower.Add(totalPower, x.GetPower(2))
	assert.Equal(int64(200), s.Schedule[2].Time)
	assert.Equal(big2exp(milliIsu), s.Schedule[2].MilliIsu)
	assert.Equal(big2exp(totalPower), s.Schedule[2].TotalPower)

	// 0.3sec
	milliIsu.Add(milliIsu, new(big.Int).Mul(totalPower, big.NewInt(100)))
	totalPower.Add(totalPower, y.GetPower(1))
	assert.Equal(int64(300), s.Schedule[3].Time)
	assert.Equal(big2exp(milliIsu), s.Schedule[3].MilliIsu)
	assert.Equal(big2exp(totalPower), s.Schedule[3].TotalPower)

	// OnSale
	assert.Contains(s.OnSale, OnSale{ItemID: 1, Time: 0})
	assert.Contains(s.OnSale, OnSale{ItemID: 2, Time: 0})
}

func TestMItem(t *testing.T) {
	assert := assert.New(t)

	item := mItem{
		ItemID: 1,
		Power1: 1,
		Power2: 2,
		Power3: 2,
		Power4: 3,
		Price1: 5,
		Price2: 4,
		Price3: 3,
		Price4: 2,
	}
	assert.Equal(item.GetPower(1).Cmp(big.NewInt(81)), 0)
	assert.Equal(item.GetPrice(1).Cmp(big.NewInt(2048)), 0)
}

func TestConv(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(Exponential{0, 0}, big2exp(str2big("0")))
	assert.Equal(Exponential{1234, 0}, big2exp(str2big("1234")))
	assert.Equal(Exponential{111111111111110, 5}, big2exp(str2big("11111111111111000000")))
}
