# qCommand

LLMを使ってpathで指定したファイルについて質問を行う

## Syntax

```bash
command q [path1], [path2]...
```

* pathについて
  * カレントディレクトリからの相対パスでTarget Codeを指定する
  * 複数指定可能
  * LLMで生成した結果は、ファイルに直接書き込まず、標準出力に出力する

* `-i`, `--input` オプションについて
  * 標準入力からテキストを受け取り、prompt.md.tmplのQuestionとして渡す
  * 入力したテキストは標準出力にも出力される
  * pオプションと併用されている場合はエラーとする
* `-p`, `--prompt` オプションについて
  * 環境変数EDITORで指定されたエディタで追加のpromptを指定できる
  * 環境変数EDITORが存在しない場合は`vi`が使われる
  * 入力したテキストはprompt.md.tmplのQuestionとして渡される
  * 入力したテキストは標準出力にも出力される
  * iオプションと併用されている場合はエラーとする
* promptに含めるknowledgeのパスの一覧を標準出力に出力する
* questionの履歴データについて
  * question毎に `プロジェクトルート/.sisho/history/questions/XXXX` フォルダを作成する（これを単体履歴フォルダと呼ぶ）
    * XXXXはKSUID
  * 履歴フォルダには以下のファイルを作成する
    * `YYYY-MM-DDTHH:MM:SS` : questionを実行した日時(ファイルは空ファイル)
    * `prompt.md` : promptの内容
      * プロンプトの組み立てが完成した直後に保存する
    * `answer.md` : promptに対する回答
* プロンプトについて
  * プロンプトはquestion/prompt.md.tmplを使って生成される
    * Targetsには指定された全てのTarget Codeの情報が入る
* knowledgeスキャンを用いてレイヤー知識リストファイル（`.knowledge.yml`）を読み込む
  * 読み込んだ直後にknowledgePathNormalizeを使ってパスを正規化する
* Target Codeに対する単一ファイル知識リストファイル（`[ファイル名].know.yml`）を読み込む
  * 読み込んだ直後にknowledgePathNormalizeを使ってパスを正規化する
* フォルダ構造情報について
  * プロジェクトコンフィグの設定に応じてフォルダ構造情報をプロンプトに追加する
  * folderStructureMakeを使う
* 使用するLLMのサービスとモデルはプロジェクトコンフィグのllmで指定できる
  * 使用するLLMのサービスとモデルの情報を標準出力に出力する
* Target Codeの一覧を標準出力に出力する