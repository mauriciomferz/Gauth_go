package gauth

import (
	"time"
)

// ...existing code...

// TimeNow is a replaceable function to get the current time (useful for testing)
var TimeNow = func() TimeFunc {
	return timeNow{}
}

type TimeFunc interface {
	Unix() int64
}

type timeNow struct{}

func (timeNow) Unix() int64 {
	return time.Now().Unix()
}
