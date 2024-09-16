package file

type Repository interface {
	Read(path string) ([]byte, error)
	Write(path string, data []byte) error
	Exists(path string) bool
	Delete(path string) error
	List(dir string) ([]string, error)
	MkdirAll(path string) error
}
