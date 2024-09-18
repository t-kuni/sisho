# extractCommand

指定したTarget Codeの内容から、知識リストファイルを抽出する

## Syntax

```bash
command extract [path]
```

* pathについて
  * カレントディレクトリからの相対パスでTarget Codeを指定する
* 指定されたpathと同階層に、知識リストファイル（`[ファイル名(拡張子除く)].know.yml`）を生成する
  * すでに同名のファイルが存在する場合は、知識リストをマージし、重複するものは１つにまとめた上で、上書きする
  * ファイルが存在しない場合は、新規作成する
* プロンプトのパラメータについて
  * Target.Path
    * Target Codeのプロジェクトルートからの相対パス
* Target Codeから知識リストを抽出する方法
  * domain/model/prompts/extract/main.goを使ってプロンプトを生成する
  * LLMにプロンプトを送信する
  * LLMの回答にCapturable Code Blockで`[ファイル名(拡張子除く)].know.yml`の内容が含まれるので、これを切り出した結果を知識リストとする


## extractKnowledgeList()

* LLMの回答から抽出した知識リストのパスはプロジェクトルートからの相対パスになっているため、指定されたpathからの相対パスに変換する