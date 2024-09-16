package file

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

type FileRepository struct{}

func NewFileRepository() *FileRepository {
	return &FileRepository{}
}

func (r *FileRepository) Read(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}

func (r *FileRepository) Write(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
}

func (r *FileRepository) Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (r *FileRepository) Delete(path string) error {
	return os.Remove(path)
}

func (r *FileRepository) List(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func (r *FileRepository) MkdirAll(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}
