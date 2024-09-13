package prompts

import (
	_ "embed"
	"html/template"
	"strings"
)

//go:embed prompt.md.tmpl
var promptTmpl string

type PromptParam struct {
	Instructions  string
	KnowledgeSets []KnowledgeSet
	Targets       []Target
}

type Target struct {
	Path    string
	Content string
}

type KnowledgeSet struct {
	Kind      string
	Knowledge []Knowledge
}

type Knowledge struct {
	Path    string
	Content string
}

func BuildPrompt(param PromptParam) (string, error) {
	tmpl, err := template.New("markdown").Parse(promptTmpl)
	if err != nil {
		return "", err
	}

	var output strings.Builder
	err = tmpl.Execute(&output, param)
	if err != nil {
		return "", err
	}

	return output.String(), nil
}

//type Answer struct {
//	Type string `json:"type" description:"\"ok\" : 完了してコード生成フェーズに移る, \"read\" : 追加のファイル読み込みを行う"`
//	Read *Read  `json:"read,omitempty"`
//}
//
//type Read struct {
//	Paths []string `json:"paths" description:"参照したいドキュメントのパス" minItems:"1" maxItems:"5" uniqueItems:"true"`
//}
//
//func (a *Answer) Parse(jsonStr string) error {
//	trimmed := pickJson(jsonStr)
//
//	err := json.Unmarshal([]byte(trimmed), &a)
//	if err != nil {
//		return err
//	}
//
//	return nil
//}
//
//func pickJson(input string) string {
//	// 先頭から最初に登場する "{" の位置を取得
//	start := strings.Index(input, "{")
//	if start == -1 {
//		return input // "{" が存在しない場合はそのまま返す
//	}
//
//	// 末尾から最後に登場する "}" の位置を取得
//	end := strings.LastIndex(input, "}")
//	if end == -1 {
//		return input // "}" が存在しない場合はそのまま返す
//	}
//
//	// 加工した文字列を返す
//	return input[start : end+1]
//}
