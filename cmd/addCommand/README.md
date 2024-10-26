# addCommand

* カレントディレクトリの.knowledges.ymlに対して、指定されたファイルを追加します。
* カレントディレクトリに.knowledges.ymlが存在しない場合は、新規に作成します。
* ファイルパスをファイルに書き込む前にutil/pathのBeforeWrite関数に掛ける

## Syntax

```bash
command add [kind] [path]
```

* path
  * カレントディレクトリからの相対パス
* kind
  * 追加するファイルの種類
  * kinds/main.go に定義されているものを指定する
