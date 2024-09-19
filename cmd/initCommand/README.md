# initCommand

カレントディレクトリをプロジェクトルートとして初期化します。

## Syntax

```bash
command init
```

* sisho.ymlを作成します。
  * すでに存在する場合は処理を中断します
* .sisho/historyフォルダを作成します
* .gitignoreに `/.sisho` を追記します
  * ファイルが存在しない場合は新規作成します

# sisho.ymlの初期値

```yml
lang: ja
#llm:
#  driver: open-ai
#  model: gpt-4-turbo
llm:
    driver: anthropic
    model: claude-3-5-sonnet-20240620
auto-collect:
    README.md: true
    "[TARGET_CODE].md": true
additional-knowledge:
    folder-structure: true
```