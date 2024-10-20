# LoadKnowledge()

* プロジェクトルートと[]Knowledgeを受け取り、[]prompts.KnowledgeSetに変換して返す
  * Knowledge.Pathのファイルを読み込み、ファイルの内容をKnowledge.Contentに設定する
  * Knowledge.Pathをプロジェクトルートからの相対パスに変換する
  * windowsの場合はパスの区切り文字を'/'に変換する
* 引数の[]KnowledgeのPathはknowledgePathNormalizeによって絶対パスに変換されている前提です