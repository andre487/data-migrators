package misc

import "time"

var epoch = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

func DateToDaysFromEpoch(dt time.Time) int64 {
	delta := dt.Sub(epoch).Hours()
	return int64(delta / 24)
}

func DaysFromEpochToDate(days int64) time.Time {
	return time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(days) * time.Hour * 24)
}
