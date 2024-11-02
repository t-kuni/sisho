//go:build windows

package path

import "strings"

func BeforeWrite(path string) string {
	return strings.ReplaceAll(path, `\`, `/`)
}

func AfterGetAbsPath(path string) (string, error) {
	return path, nil
}
