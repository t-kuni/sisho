# Software Architecture

* /cmd/main.go でサブコマンドの初期化、依存モジュールの初期化とDIを行う
* /cmd/**/main.go に各サブコマンドの処理を記述する。ドメインロジックは大枠ここに記述される
* /domain 配下にドメインモデルやビジネスロジック、infrastructure層とのインターフェースを定義する
* /infrastructure 配下に外部との通信やファイルへのアクセスなどのI/O処理を定義する

# Coding Rules

* エラーをリターンする場合はgithub.com/rotisserie/erisを使ってwrapする
* 別のモジュールから使われる可能性のあるものにはコメントを付ける（関数ドキュメントや構造体のフィールドコメントなど）
* // FIXME や // TODO 、 // NOTE などの注釈コメントを勝手に消さないこと

# エントリーポイント

当該ソフトウェアのエントリーポイントは /main.go です。

# プロジェクトコンフィグについて

* `sisho.yml`のこと

## プロジェクトコンフィグ(sisho.yml)のサンプル

```yaml
lang: ja
llm:
  driver: open-ai
  model: gpt-4o
auto-collect:
  README.md: true
  "[TARGET_CODE].md": true
additional-knowledge:
  folder-structure: true
tasks:
  - name: build
    run: |
      cd src
      npm run build
```

## llmについて

* driver
  * `open-ai` を指定した場合、OpenAIのAPIを利用する
  * `anthropic` を指定した場合、AnthropicのAPIを利用する
* model
  * string型
  * 各種サービスのモデル名に準拠

## auto-collectについて

* README.md
  * bool型 
  * trueの場合、コンテキストスキャンを用いて、各階層のREADME.mdをknowledgeとしてLLMに提示する
  * kindはspecificationsとして扱う
* "[TARGET_CODE].md"
  * bool型
  * trueの場合、Target Codeと同階層の[TARGET_CODE].mdをknowledgeとしてLLMに提示する
  * 例えば、Target Codeが`aaa/bbb/main.go` の場合、`aaa/bbb/main.go.md` が対象となる 
    * ただし、 `aaa/main.go.md` は対象外なので注意
  * kindはspecificationsとして扱う

## additional-knowledge.folder-structureについて

* bool型
* trueの場合、makeコマンド実行時、プロジェクトルート配下のフォルダ構造情報をプロンプトに追加する

## tasksについて

* タスクを定義します
* 主にビルドやテストコードの実行を定義します
* 用途
  * fix:task サブコマンドで使用します
* フィールドについて
  * name
    * タスク名
  * run
    * タスクの実行コマンド

# プロジェクトルートとは

プロジェクトルートは`sisho.yml`が存在するディレクトリを指します。

# Target Codeとは

本ツールでスキャフォルドする対象のコードを指します。
makeコマンドの引数で指定します。

# コンテキストスキャンとは

* プロジェクトルートからTarget Codeのディレクトリまでの各階層を走査し必要な処理を行うことです。
  * 隠しフォルダは対象外です。
* 用途の例
  * 各階層の.knowledge.ymlを読み込む
  * 各階層のREADME.mdを読み込む

# プロジェクトスキャンとは

* プロジェクトルート以下の全ての階層を走査し必要な処理を行うことです。
  * 隠しフォルダは対象外です。

# 知識リストファイルとは

* ２種類ある
  * レイヤー知識リストファイル（`.knowledge.yml`）
    * このファイルが存在するディレクトリ配下（再帰的）に有効な知識リスト
  * 単一ファイル知識リストファイル（`[ファイル名].know.yml`）
    * [ファイル名]のファイルに対して有効な知識リスト
    * 例えば `main.go` ならば `main.go.know.yml` となる
* Target Codeに対応する知識リストファイルを読み込み、makeコマンド実行時にLLMに提示するファイルを指定します。
* このファイルは省略可能です。

## 知識リストファイルのサンプル

```yaml
knowledge:
  - path: go.mod
    kind: dependencies
  - path: cmd/makeCommand/main.go
    kind: examples
  - path: lib/package1/main.go
    kind: implementations
    chain-make: true
  - path: lib/package2/main.go
    kind: implementations
    chain-make: true
  - path: README.md
    kind: specifications
```

* path
  * string型
  * Syntax
    * 相対パス指定
      * 例： `makeCommand/main.go`, `../lib/package1/main.go`
      * 説明： 当該ファイルからの相対パスを指定します
    * 絶対パス指定
      * 例： `/Users/xxx/project/cmd/makeCommand/main.go`
      * 説明： システムのルートからの絶対パスを指定します
    * プロジェクトルートからの相対パス指定
      * 例： `@/cmd/makeCommand/main.go`
      * 説明： プロジェクトルートからの相対パスを指定します
  * 当該.knowledge.ymlから対象ファイルまでの相対パスを指定します
* chain-make
  * bool型
  * 省略可能。省略した場合、falseとして扱われます

# knowledgeスキャンとは

* コンテキストスキャンを用いて各階層のレイヤー知識リストファイル（`.knowledge.yml`）を読み込むことです。
  * `.knowledge.yml`は省略可能なので、存在しない場合は無視され処理は継続します。
  * `.knowledge.yml`で指定したファイルが重複する場合は１つにまとめられます。
  * Target Codeが複数指定された場合、全てのTarget CodeについてKnowledgeスキャンを行います

# Capturable Code Block とは

以下の書式に従うコードブロックを指します。

* コードブロック開始の書式： \<!-- CODE_BLOCK_BEGIN -->```[Target Code Path]
* コードブロック終了の書式： ```\<!-- CODE_BLOCK_END -->
* [Target Code Path]はTarget Codeのパス（プロジェクトルートからの相対パス）を指します
* Capturable Code Blockの内容を取得する場合はextractCodeBlockサービスを利用します

# フォルダ構造情報とは

* treeコマンドの出力のようなフォルダ構造を表したテキストのこと
* 隠しフォルダは出力されません
* フォルダの名前には接頭辞`/`がつきます
* .sishoignoreファイルに記載されたファイル・フォルダは出力されません

## フォルダ構造情報のサンプル

```txt
A
  /B
    a.txt
  /C
    b.go
  /c.md
D
  d.yml
```

# 依存グラフ(depsGraph)とは

* 指定したファイルに依存しているファイルを逆引きするためのグラフです

# .sishoignoreファイルとは

* プロジェクトルートに配置する
* パースは github.com/denormal/go-gitignore を利用する