# ScanKnowledge()

* knowledgeスキャンを行う
* プロジェクトルートのlパスとTarget Codeのパスの配列を受け取る
* Knowledge.Pathはプロジェクトルートからの相対パスに置き換える
* Knowledge.Pathが重複する場合は１つにまとめる
* 同時に単一ファイル知識リストファイル（`[ファイル名(拡張子除く)].know.yml`）も読み込む