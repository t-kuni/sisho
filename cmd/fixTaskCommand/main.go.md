# makeCommand

タスクを実行し、エラーが発生した場合、LLMを使って修正対象のパスを抽出し、修正を行う



## Syntax

```bash
command fix:task [taskName]
```

* パラメータ
  * taskName
* 処理概要
  1. 試行ループ（tオプションで指定した回数繰り返し）
     1. taskNameを用いてsisho.ymlに定義されたコマンド情報を取得
     2. タスクのrunに定義されたコマンドを実行する（同一プロセス）
        1. すべてのコマンドが正常完了した場合はそのまま終了する
     3. エラーが発生した場合、標準出力と標準エラー出力を取得する
     4. 手順3で取得した文字列からdomain/model/chatとdomain/model/prompts/extractPathsを使って修正対象のパスを抽出する
     5. 修正対象のパスが存在しない場合はエラーとする
     6. タスクのエラーメッセージと、修正対象のパスをmakeServiceに渡して修正を行う
* ファイルを生成する処理はmakeServiceを使って行う
* オプションについて
    * `-t`, `--try` オプション
      * 試行回数
      * デフォルト：1
    * '-d', '--dry-run' オプションについて
      * service/makeの引数dryRunに渡す
* 履歴データについて
    * fix:task毎に `プロジェクトルート/.sisho/fixTask/XXXX` フォルダを作成する
        * XXXXはKSUID
    * 履歴フォルダには以下のファイルを作成する
        * `YYYY-MM-DDTHH-MM-SS` : makeを実行した日時(ファイルは空ファイル)
        * `prompt_XX.md` : promptの内容(XXは1から始まる連番)
            * プロンプトの組み立てが完成した直後に保存する
        * `answer_XX.md` : promptに対する回答(XXは1から始まる連番)