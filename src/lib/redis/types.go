package lib_redis

import ()

type SortedSetMember struct {
	Name  string
	Score uint64
}

type GeoSearchLocation struct {
	Name    string
	Lon     float64
	Lat     float64
	Dist    float64
	GeoHash int64
}
