package utils

import "time"

func Now() time.Time {
	return time.Now()
}

func NowUnixNano() int64 {
	return Now().UnixNano()
}
