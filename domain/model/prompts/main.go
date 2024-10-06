package prompts

import (
	_ "embed"
	"strings"
	"text/template"
)

//go:embed prompt.md.tmpl
var promptTmpl string

type PromptParam struct {
	Instructions    string
	KnowledgeSets   []KnowledgeSet
	Targets         []Target
	FolderStructure string
	GeneratePath    string
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
