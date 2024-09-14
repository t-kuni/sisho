# LLMでスキャフォルドするツール

```
# Install
go install github.com/t-kuni/sisho

# Initialize project
sisho init

# Add knowledge to generate code
# Syntax: sisho add [kind] [path]
# Example:
sisho add specifications swagger.yml
sisho add specifications er.mmd
sisho add examples handlers/getUser.go
sisho add implementations handlers/getUser.go
sisho add dependencies go.mod

# Code generation
# Syntax: sisho make [target path1] [target path2] ... 
export ANTHROPIC_API_KEY="xxxx"
sisho make handlers/postUser.go
```

# Development

```
go run main.go make [target path]
```

## sisho.ymlのサンプル

```yaml
lang: ja
auto-collect:
  README.md: true
  "[TARGET_CODE].md": true
```

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

## プロジェクトルートとは

プロジェクトルートは`sisho.yml`が存在するディレクトリを指します。

## Target Codeとは

本ツールでスキャフォルドする対象のコードを指します。
makeコマンドの引数で指定します。

## .knowledge.ymlとは

* makeコマンド実行時にLLMに提示するファイルを指定するためのファイルです。

## Targetスキャンとは

* プロジェクトルートからTarget Codeのディレクトリまでの各階層を走査し必要な処理を行うことです。
* 用途の例
  * 各階層の.knowledge.ymlを読み込む
  * 各階層のREADME.mdを読み込む

### .knowledge.ymlのサンプル

```yaml
knowledge:
  - path: go.mod
    kind: dependencies
  - path: cmd/makeCommand/main.go:38
    kind: examples
  - path: kinds/main.go
    kind: implementations
  - path: README.md
    kind: specifications
```

### knowledgeスキャンとは

* コンテキストスキャンを用いて各階層の`.knowledge.yml`を読み込むことです。
  * `.knowledge.yml`は省略可能なので、存在しない場合は無視され処理は継続します。
  * `.knowledge.yml`で指定したファイルが重複する場合は１つにまとめられます。
  * Target Codeが複数指定された場合、全てのTarget CodeについてKnowledgeスキャンを行います