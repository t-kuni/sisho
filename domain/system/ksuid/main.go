//go:generate mockgen -source=$GOFILE -destination=${GOFILE}_mock.go -package=$GOPACKAGE

package ksuid

type IKsuid interface {
	New() string
}
