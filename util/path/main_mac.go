//go:build darwin

package path

import "path/filepath"

func BeforeWrite(path string) string {
	return path
}

func AfterGetAbsPath(path string) (string, error) {
	// macの場合、絶対パスを取得したときに、/var/folder/... と /private/var/folder/... と同じフォルダを指す２種類のパスが取得できてしまう
	// 以下の処理を入れることで /private に統一できる
	return filepath.EvalSymlinks(path)
}
