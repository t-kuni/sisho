* 実行タイミング
  * PRを作成時、PRにコミットが追加された時
* 実行環境
  * Ubuntu
  * Windows
  * macOS
* Goバージョンはgo.modに記載されているバージョンを使用
* 実行内容
  * go install go.uber.org/mock/mockgen@latest
  * go generate ./...
  * go test ./...