package main

import (
	"log"
	"math"
	"math/big"
	"math/rand"
	"sync"
	"time"
)

var (
	utilMtx  sync.Mutex
	big10    = big.NewInt(10)
	e10cache = make(map[int64]*big.Int)
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func str2big(s string) *big.Int {
	x := new(big.Int)
	x.SetString(s, 10)
	return x
}

func big2exp(n *big.Int) Exponential {
	if n == nil {
		log.Fatalln("big2expB : nil")
	} else {
		sign := n.Sign()
		if sign < 0 {
			log.Fatalln("big2expB : negative")
		} else {
			if n.IsInt64() {
				v := n.Int64()
				if v < 1000000000000000 { // 10^15
					return Exponential{v, 0}
				}
			}
			x := int64(math.Floor(float64(n.BitLen()-1)*0.301029995663981 + 1))
			i := x - 17
			if i <= 0 {
				i = 1
			}

			utilMtx.Lock()
			e, ok := e10cache[i]
			utilMtx.Unlock()

			if !ok {
				e = new(big.Int).Exp(big10, big.NewInt(i), nil)

				utilMtx.Lock()
				e10cache[i] = e
				utilMtx.Unlock()
			}

			t := new(big.Int).Quo(n, e)
			for {
				if t.IsInt64() {
					y := t.Int64()
					if y < 100000000000000 { // 10^14
						log.Fatalln("big2expB : something wrong", y, n.String(), x, i)
					}
					if y < 1000000000000000 { // 10^15
						return Exponential{y, i}
					}
				}
				t.Quo(t, big10)
				i += 1
			}
		}
	}
	panic("something wrong")
}

func exp2big(e Exponential) *big.Int {
	n := big.NewInt(e.Mantissa)
	n.Exp(n, big.NewInt(e.Exponent), nil)
	return n
}

func unixMilliSecond(t time.Time) int64 {
	return t.UnixNano() / 1000000
}

func genRandomNumberString(length int) string {
	runes := []rune("0123456789")

	b := make([]rune, length)
	for i := range b {
		b[i] = runes[rand.Intn(len(runes))]
	}
	if b[0] == '0' {
		b[0] = runes[rand.Intn(len(runes)-1)+1]
	}
	return string(b)
}
