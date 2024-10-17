# makeCommand

LLMを使ってpathで指定したファイルを生成する

## Syntax

```bash
command make [path1], [path2]...
```

* pathについて
  * カレントディレクトリからの相対パスでTarget Codeを指定する
  * 複数指定可能

* ファイルを生成する処理はmakeServiceを使って行う
* オプションについて
  * `-c`, `--chain` オプション
    * 連鎖的生成を行う
  * `-i`, `--input` 
    * 標準入力からテキストを受け取り、prompt.md.tmplのInstructionsとして渡す
    * pオプションと併用されている場合はエラーとする
  * `-p`, `--prompt` オプションについて
    * 環境変数EDITORで指定されたエディタで追加のpromptを指定できる
    * 環境変数EDITORが存在しない場合は`vi`が使われる
    * 入力したテキストはprompt.md.tmplのInstructionsとして渡される
    * iオプションと併用されている場合はエラーとする
  * `-a`, `--apply` オプションについて
    * LLMの出力をファイルに反映します 
  * '-d', '--dry-run' オプションについて
    * service/makeの引数dryRunに渡す