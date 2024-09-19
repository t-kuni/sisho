//go:generate mockgen -source=$GOFILE -destination=${GOFILE}_mock.go -package=$GOPACKAGE

package file

type Repository interface {
	Getwd() (string, error)
}
