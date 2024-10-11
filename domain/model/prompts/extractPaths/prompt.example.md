以下のコマンド実行結果から、エラーが発生しているファイルのパスを抽出してください。
警告は無視してください。
プロジェクトルートからの相対パスを返してください。

# 実行したコマンド

```sh
(>&2 echo "エラーメッセージ") && exit 1

```

# コマンド実行結果（標準ストリーム）

```txt
Stdout:

Stderr:
Error: Build failed due to syntax error in source files.

Syntax error in file: /home/user/project/src/main.cpp, line 45
Expected ';' before '}' token

File not found: /home/user/project/src/unknown.cpp
Referenced by: /home/user/project/src/module.cpp

Warning: Deprecated function used in /home/user/project/src/utils.cpp, line 78
Consider replacing with 'new_function()' for better performance.

Please review and fix the syntax errors and warnings in the mentioned files.

Error occurred during the execution of 'make build' command.
Exit code: 1

Build terminated unexpectedly. Resolve the issues and try running the build again.


Error:
exit status 1
```

# Folder Structure

```
project
├── include
│   └── utils.h
├── src
│   ├── main.cpp
│   ├── module.cpp
│   ├── helpers.cpp
│   ├── utils.cpp
│   └── random_file.cpp
├── build
│   ├── output.o
│   └── temp
│       └── temp_file.o
└── docs
    └── README.md

```

# Capturable Code Block とは

以下の書式に従うコードブロックを指します。

* コードブロック開始の書式： \<!-- CODE_BLOCK_BEGIN -->```json
* コードブロック終了の書式： ```\<!-- CODE_BLOCK_END -->

# Answer Syntax

```yaml
type: array
description: 修正が必要なパスのリスト（プロジェクトルートからの相対パス）
items:
  x-stoplight:
    id: v421jsbv91ti7
  type: string
```

# Answer

* 説明は省略します。
* コードブロックは Capturable Code Block に従って記載します。


