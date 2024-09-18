# initCommand

カレントディレクトリをプロジェクトルートとして初期化します。

## Syntax

```bash
command init
```

* sisho.ymlを作成します。
  * すでに存在する場合は処理を中断します
* .sisho/historyフォルダを作成します
* .gitignoreに `/.sisho` を追記します
  * ファイルが存在しない場合は新規作成します