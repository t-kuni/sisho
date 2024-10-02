package extract

import (
	"bytes"
	_ "embed"
	"github.com/t-kuni/sisho/domain/model/prompts"
	"path/filepath"
	"text/template"
)

//go:embed prompt.md.tmpl
var promptTmpl string

type PromptParam struct {
	Target            prompts.Target
	FolderStructure   string
	KnowledgeListPath string
}

func BuildPrompt(param PromptParam) (string, error) {
	tmpl, err := template.New("markdown").Parse(promptTmpl)
	if err != nil {
		return "", err
	}

	// Generate the knowledge list file path
	fileName := filepath.Base(param.Target.Path)
	param.KnowledgeListPath = filepath.Join(filepath.Dir(param.Target.Path), fileName+".know.yml")

	var output bytes.Buffer
	err = tmpl.Execute(&output, param)
	if err != nil {
		return "", err
	}

	return output.String(), nil
}
