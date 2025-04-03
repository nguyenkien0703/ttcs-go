package lib_math

import "math"

func Deg2Rad(deg float64) float64 {
	return deg * math.Pi / 180
}
func Rad2Deg(rad float64) float64 {
	return rad * 180 / math.Pi
}

func SafeAddUint32(src uint32, add uint32) uint32 {
	if (math.MaxUint32 - src) < add {
		return math.MaxUint32
	}
	return src + add
}

func SafeAddUint64(src uint64, add uint64) uint64 {
	if (math.MaxUint64 - src) < add {
		return math.MaxUint64
	}
	return src + add
}

func SafeSubUint32(src uint32, sub uint32) uint32 {
	if sub > src {
		return 0
	}
	return src - sub
}

func SafeSubUint64(src uint64, sub uint64) uint64 {
	if sub > src {
		return 0
	}
	return src - sub
}

func SafeAddInt64(src int64, add int64) int64 {
	if 0 < add && (math.MaxInt64-add) < src {
		return math.MaxInt64
	} else if add < 0 && src < (math.MinInt64-add) {
		return math.MinInt64
	}
	return src + add
}
