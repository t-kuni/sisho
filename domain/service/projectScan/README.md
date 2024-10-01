# scan()

* プロジェクトスキャンを実装する
* プロジェクトルートの `.sishoignore` を読み込み、ここに記載されているパスはスキャン対象から除外する
  * github.com/denormal/go-gitignore を使用する
* 各階層での処理本体は引数のクロージャにて実装する