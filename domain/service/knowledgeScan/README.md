# ScanKnowledgeMultipleTarget()

* プロジェクトルートのパスとTarget Codeのパスの配列を受け取る
* ScanKnowledge()を用いてknowledgeスキャンを行う
* 最後に、Knowledge.Pathが重複する場合は１つにまとめる

# ScanKnowledge()

* knowledgeスキャンを行う
  * .knowledge.ymlを１つ読み込む毎にknowledgePathNormalizeを用いてKnowledge.Pathを絶対パスに変換する
* プロジェクトルートのパスとTarget Codeのパスを受け取る
* knowledgePathNormalizeを用いてKnowledge.Pathを絶対パスに変換する
* 同時に単一ファイル知識リストファイル（`[ファイル名(拡張子除く)].know.yml`）も読み込む
  * 読み込んだ直後にknowledgePathNormalizeを用いてKnowledge.Pathを絶対パスに変換する
* 最後に、Knowledge.Pathが重複する場合は１つにまとめる