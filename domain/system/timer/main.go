//go:generate mockgen -source=$GOFILE -destination=${GOFILE}_mock.go -package=$GOPACKAGE

package timer

import "time"

type ITimer interface {
	Now() time.Time
}
