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

* 生成ループとは 
  * 複数のTarget Codeが指定された場合、それぞれのTarget Codeに対して以下の処理を行うこと
    1. Target Codeを全て読み込み（生成毎に最新のTarget Codeを読み込みたいためループ毎に読み込む）
    2. 知識ファイルの重複を排除する
    3. 生成ターゲットの知識リストファイル収集
    4. プロンプト組み立て
    5. LLMに送信
    6. 回答をTarget Codeに反映
* 生成ターゲットとは
  * 生成ループの各ループで、生成する対象となるTarget Codeのこと
* promptはprompts/prompt.md.tmplに従う
* `-c`, `--chain` オプションについて
  * 指定されたTarget Codeに依存しているファイルを依存グラフ（.sisho/deps-graph.json）から再帰的に取得し、それらのファイルもTarget Codeとして扱う
  * Target Codeの順番は依存グラフの深度の浅い順に並べる
  * deps-graph.jsonが存在しない場合はエラーを出力する
  * deps-graph.jsonにTarget Codeのパスが存在しない場合は一番深い深度のTarget Codeとして扱う
* `-i`, `--input` オプションについて
  * 標準入力からテキストを受け取り、prompt.md.tmplのInstructionsとして渡す
  * 入力したテキストは標準出力にも出力される
  * pオプションと併用されている場合はエラーとする
* `-p`, `--prompt` オプションについて
  * 環境変数EDITORで指定されたエディタで追加のpromptを指定できる
  * 環境変数EDITORが存在しない場合は`vi`が使われる
  * 入力したテキストはprompt.md.tmplのInstructionsとして渡される
  * 入力したテキストは標準出力にも出力される
  * iオプションと併用されている場合はエラーとする
* `-a`, `--apply` オプションについて
  * LLMの出力をファイルに反映します 
    * LLMの出力には余分な文章が含まれる可能性があるため、 Capturable Code Blockの仕様に基づいて切り出した結果をファイルに反映します
  * 標準出力には反映したファイルのパスと差分を出力します。
* promptに含めるknowledgeのパスの一覧を標準出力に出力する
* makeの履歴データについて
  * make毎に `プロジェクトルート/.sisho/history/XXXX` フォルダを作成する（これを単体履歴フォルダと呼ぶ）
    * XXXXはKSUID
  * 履歴フォルダには以下のファイルを作成する
    * `YYYY-MM-DDTHH:MM:SS` : makeを実行した日時(ファイルは空ファイル)
    * `prompt_XX.md` : promptの内容(XXは1から始まる連番)
      * プロンプトの組み立てが完成した直後に保存する
    * `answer_XX.md` : promptに対する回答(XXは1から始まる連番)
* プロンプトについて
  * プロンプトはdomain/model/prompts/prompt.md.tmplを使って生成される
    * Targetsには指定された全てのTarget Codeの情報が入る
  * Target Codeが複数存在する場合、毎回プロンプトを作り直す
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
* 生成ターゲット毎にセパレーターを標準出力に出力する