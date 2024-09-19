package file

import (
	"os"
)

// FileRepository handles file operations.
type FileRepository struct{}

// NewFileRepository creates a new instance of FileRepository.
func NewFileRepository() *FileRepository {
	return &FileRepository{}
}

// Getwd returns the current working directory.
func (r *FileRepository) Getwd() (string, error) {
	return os.Getwd()
}
