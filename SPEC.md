## Software Architecture

* /cmd/main.go でサブコマンドの初期化、依存モジュールの初期化とDIを行う
* /cmd/**/main.go に各サブコマンドの処理を記述する。ドメインロジックは大枠ここに記述される
* /domain 配下にドメインモデルやビジネスロジック、infrastructure層とのインターフェースを定義する
* /infrastructure 配下に外部との通信やファイルへのアクセスなどのI/O処理を定義する

## エントリーポイント

当該ソフトウェアのエントリーポイントは /main.go です。

## プロジェクトコンフィグについて

* `sisho.yml`のこと

### プロジェクトコンフィグ(sisho.yml)のサンプル

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
```

### llmについて

* driver
  * `open-ai` を指定した場合、OpenAIのAPIを利用する
  * `anthropic` を指定した場合、AnthropicのAPIを利用する
* model
  * string型
  * 各種サービスのモデル名に準拠

### auto-collectについて

* README.md
  * bool型 
  * trueの場合、コンテキストスキャンを用いて、各階層のREADME.mdをknowledgeとしてLLMに提示する
  * kindはspecificationsとして扱う
* "[TARGET_CODE].md"
  * bool型
  * trueの場合、コンテキストスキャンを用いて、各階層の[TARGET_CODE].mdをknowledgeとしてLLMに提示する
  * [TARGET_CODE]はTarget Codeのファイル名を指す
  * kindはspecificationsとして扱う

### additional-knowledge.folder-structureについて

* bool型
* trueの場合、makeコマンド実行時、プロジェクトルート配下のフォルダ構造情報をプロンプトに追加する

## プロジェクトルートとは

プロジェクトルートは`sisho.yml`が存在するディレクトリを指します。

## Target Codeとは

本ツールでスキャフォルドする対象のコードを指します。
makeコマンドの引数で指定します。

## コンテキストスキャンとは

* プロジェクトルートからTarget Codeのディレクトリまでの各階層を走査し必要な処理を行うことです。
  * 隠しフォルダは対象外です。
* 用途の例
  * 各階層の.knowledge.ymlを読み込む
  * 各階層のREADME.mdを読み込む

## プロジェクトスキャンとは

* プロジェクトルート以下の全ての階層を走査し必要な処理を行うことです。
  * 隠しフォルダは対象外です。

## 知識リストファイルとは

* ２種類ある
  * レイヤー知識リストファイル（`.knowledge.yml`）
    * このファイルが存在するディレクトリ配下（再帰的）に有効な知識リスト
  * 単一ファイル知識リストファイル（`[ファイル名].know.yml`）
    * [ファイル名]のファイルに対して有効な知識リスト
    * 例えば `main.go` ならば `main.go.know.yml` となる
* Target Codeに対応する知識リストファイルを読み込み、makeコマンド実行時にLLMに提示するファイルを指定します。
* このファイルは省略可能です。

### 知識リストファイルのサンプル

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
  * 当該.knowledge.ymlから対象ファイルまでの相対パスを指定します
* chain-make
  * bool型
  * 省略可能。省略した場合、falseとして扱われます

## knowledgeスキャンとは

* コンテキストスキャンを用いて各階層のレイヤー知識リストファイル（`.knowledge.yml`）を読み込むことです。
  * `.knowledge.yml`は省略可能なので、存在しない場合は無視され処理は継続します。
  * `.knowledge.yml`で指定したファイルが重複する場合は１つにまとめられます。
  * Target Codeが複数指定された場合、全てのTarget CodeについてKnowledgeスキャンを行います

## Capturable Code Block とは

以下の書式に従うコードブロックを指します。

* コードブロック開始の書式： \<!-- CODE_BLOCK_BEGIN -->```[Target Code Path]
* コードブロック終了の書式： ```\<!-- CODE_BLOCK_END -->
* [Target Code Path]はTarget Codeのパス（プロジェクトルートからの相対パス）を指します

### Capturable Code Blockの本文を切り出す正規表現

```go
re := regexp.MustCompile("(?s)(\n|^)<!-- CODE_BLOCK_BEGIN -->```" + regexp.QuoteMeta(path) + "(.*)```.?<!-- CODE_BLOCK_END -->(\n|$)")
```

## フォルダ構造情報とは

* treeコマンドの出力のようなフォルダ構造を表したテキストのこと
* 隠しフォルダは出力されません
* フォルダの名前には接頭辞`/`がつきます

### フォルダ構造情報のサンプル

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
