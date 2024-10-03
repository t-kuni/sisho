# makeTree()

* 引数
  * プロジェクトルートのパス
* 戻り値
  * フォルダ構成情報（文字列）
* フォルダ構成情報を作成して返します
* .sishoignoreにマッチするファイルやフォルダはプロンプトに含めない
  * .sishoignoreが存在しない場合は.sishoignoreに基づくスキップは行わない
  * .gitignoreは考慮しなくてよい
* filepath.Walkを使う