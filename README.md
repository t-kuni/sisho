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

## sisho.yml\のサンプル

```
lang: ja
```

## プロジェクトルートとは

プロジェクトルートは`sisho.yml`が存在するディレクトリを指します。

## Target Codeとは

本ツールでスキャフォルドする対象のコードを指します。
makeコマンドの引数で指定します。

## .knowledge.ymlとは

* makeコマンド実行時にLLMに提示するファイルを指定するためのファイルです。

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

* プロジェクトルートからTarget Codeのディレクトリまでの各階層の`.knowledge.yml`を読み込むことです。
  * `.knowledge.yml`は省略可能なので、存在しない場合は無視され処理は継続します。
  * `.knowledge.yml`で指定したファイルが重複する場合は１つにまとめられます。
  * Target Codeが複数指定された場合、全てのTarget CodeについてKnowledgeスキャンを行います