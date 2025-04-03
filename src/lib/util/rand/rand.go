package lib_util_rand

import (
	"github.com/seehuhn/mt19937"
	"math/rand"
)

type Random struct {
	*rand.Rand
}

func NewRandom(seed uint64) *Random {
	r := rand.New(mt19937.New())
	r.Seed(int64(seed))
	return &Random{
		Rand: r,
	}
}
func (self *Random) Ints(min, max int) int {
	if max <= min {
		return min
	}
	return self.Intn(max-min+1) + min
}
func (self *Random) Int64s(min, max int64) int64 {
	if max <= min {
		return min
	}
	return self.Int63n(max-min+1) + min
}
