package eventsource

import (
	"strconv"
	"time"
)

type EpochMillis int64

func (e EpochMillis) Int64() int64 {
	return int64(e)
}

func (e EpochMillis) String() string {
	return strconv.FormatInt(int64(e), 10)
}

func (e EpochMillis) Time() time.Time {
	return time.UnixMilli(e.Int64())
}

func Now() EpochMillis {
	return Time(time.Now())
}

func Time(t time.Time) EpochMillis {
	return EpochMillis(t.UnixMilli())
}
