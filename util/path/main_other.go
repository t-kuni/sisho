//go:build !windows

package path

func BeforeWrite(path string) string {
	return path
}
