package question

import (
	_ "embed"
	"github.com/t-kuni/sisho/domain/model/prompts"
	"strings"
	"text/template"
)

//go:embed prompt.md.tmpl
var promptTmpl string

type PromptParam struct {
	Question        string
	KnowledgeSets   []prompts.KnowledgeSet
	Targets         []prompts.Target
	FolderStructure string
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
