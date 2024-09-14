# makeCommand

LLMを使ってpathで指定したファイルを生成する

## Syntax

```bash
command make [path1], [path2]...
```

* pathについて
  * カレントディレクトリからの相対パスでTarget Codeを指定する
  * 複数指定可能
  * LLMで生成した結果は、ファイルに直接書き込まず、標準出力に出力する

* promptはprompts/prompt.md.tmplに従う
* `-p`, `--prompt` オプションについて
  * 環境変数EDITORで指定されたエディタで追加のpromptを指定できる
  * 環境変数EDITORが存在しない場合は`vi`が使われる
  * 入力したテキストはprompt.md.tmplのInstructionsとして渡される
* promptに含めるknowledgeのパスの一覧を標準出力に出力する
* makeの履歴データについて
  * make毎に `プロジェクトルート/.sisho/history/XXXX` フォルダを作成する（これを単体履歴フォルダと呼ぶ）
    * XXXXはKSUID
  * 履歴フォルダには以下のファイルを作成する
    * `YYYY-MM-DDTHH:MM:SS` : makeを実行した日時(ファイルは空ファイル)
    * `prompt.md` : promptの内容
    * `answer.md` : promptに対する回答
* プロンプトについて
  * 指定したTarget Code１つにつき、LLMとのやり取りの往復が１回発生する
  * １回めのプロンプトはprompts/prompt.md.tmplを使って生成される
  * ２回め以降のプロンプトはprompts/oneMoreMake.md.tmplを使って生成される
* knowledgeスキャンを行う