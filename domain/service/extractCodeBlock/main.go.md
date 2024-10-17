# extractCodeBlock

* 以下の開始位置から終了位置までのテキストを切り出す
	* 開始位置： 行頭に出現する ```\<!-- CODE_BLOCK_BEGIN -->[ファイルパス]
		* [ファイルパス]は引数で受け取る
	* 終了位置：  行頭に出現する ```\<!-- CODE_BLOCK_END -->
      * ``` と \<!-- CODE_BLOCK_END --> の間に改行が入るケースがある
* 終了位置は開始位置より後ろであること
* 終了位置に該当するものが複数ある場合は最も開始位置に近い方を選ぶ
* 引数のファイルパスに対応するコードブロックが見つからない場合はエラーを返す
* コードブロックが見つかったが、開始位置と終了位置の間にコンテンツがない場合は空文字を返す
* 正規表現は使わないで実装する