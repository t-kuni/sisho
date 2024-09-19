package timer

import "time"

type Timer struct{}

func NewTimer() *Timer {
	return &Timer{}
}

func (t *Timer) Now() time.Time {
	return time.Now()
}
