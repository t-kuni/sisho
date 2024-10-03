# depsGraphCommand

プロジェクトスキャンを行い、単一ファイル知識リストファイルを読み込み、依存グラフを生成して保存します。

## Syntax

```bash
command deps-graph
```

# 処理概要

* プロジェクトスキャンを用いて、単一ファイル知識リストファイルを読み込みます
* スキャンの進捗を標準出力に表示します
* 単一ファイル知識リストファイルのknowledgeのうち、`chain-make`がtrueのものを集めます(kindは不問)
  * 単一ファイル知識リストファイルのTarget CodeがDependant, knowledgeのpathがDependencyです
* 依存グラフ変換を行います
  * 入力
    * [](Dependent, Dependency)
    * Dependent, Dependencyともにプロジェクトルートからの相対パス
  * 出力
    * map[Dependency][]Dependent
* 依存グラフを `.sisho/deps-graph.json` (プロジェクトルートからの相対パス) に保存します
