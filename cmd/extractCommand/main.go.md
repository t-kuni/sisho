# extractCommand

指定したTarget Codeの内容から、知識リストファイルを抽出する

## Syntax

```bash
command extract [path]
```

* pathについて
  * カレントディレクトリからの相対パスでTarget Codeを指定する
* 指定されたpathと同階層に、知識リストファイル（`[ファイル名].know.yml`）を生成する
  * すでに同名のファイルが存在する場合は、知識リストをマージし、重複するものは１つにまとめた上で、上書きする
  * ファイルが存在しない場合は、新規作成する
* プロンプトのパラメータについて
  * Target.Path
    * Target Codeのプロジェクトルートからの相対パス
* Target Codeから知識リストを抽出する方法
  * domain/model/prompts/extract/main.goを使ってプロンプトを生成する
  * LLMにプロンプトを送信する
  * LLMの回答にCapturable Code Blockで`[ファイル名].know.yml`の内容が含まれるので、これを切り出した結果を知識リストとする
  * LLMの回答から抽出した知識リストのパスはプロジェクトルートからの相対パスになっている（@表記ではない）
    * filepath.Cleanに掛けて、先頭に `@/` を付与して @表記に変換して保存する
* 知識リストの重複チェックは knowledgePathNormalize で正規化したパス同士で比較する（この正規化したパスは保存には使わない）
* フォルダ構造情報をプロンプトに追加する
  * folderStructureMakeを使う
* 生成が途中で終了した場合はエラー扱いとして、その理由を標準出力に出力する