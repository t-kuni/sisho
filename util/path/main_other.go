//go:build !windows && !darwin

package path

func BeforeWrite(path string) string {
	return path
}

func AfterGetAbsPath(path string) (string, error) {
	return path, nil
}
