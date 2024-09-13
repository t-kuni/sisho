package oneMoreMake

import (
	_ "embed"
	"html/template"
	"strings"
)

//go:embed prompt.md.tmpl
var promptTmpl string

type PromptParam struct {
	Path string
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
