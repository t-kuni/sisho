package extractCodeBlock

import (
	"fmt"
	"regexp"
	"strings"
)

type CodeBlockExtractService struct {
}

func NewCodeBlockExtractService() *CodeBlockExtractService {
	return &CodeBlockExtractService{}
}

func (s *CodeBlockExtractService) ExtractCodeBlock(inputText, filePath string) (string, error) {
	startRegex := regexp.MustCompile(fmt.Sprintf("(^|\n)<!-- CODE_BLOCK_BEGIN -->```%s", regexp.QuoteMeta(filePath)))
	endRegex := regexp.MustCompile("(^|\n)```(.|\n)?<!-- CODE_BLOCK_END -->")

	startMatched := startRegex.FindStringIndex(inputText)
	if startMatched == nil {
		return "", fmt.Errorf("code block not found for file path: %s", filePath)
	}

	contentStart := startMatched[1] + 1 // +1 for the newline after the marker

	endMatched := endRegex.FindStringIndex(inputText[contentStart:])
	if endMatched == nil {
		return "", fmt.Errorf("end of code block not found for file path: %s", filePath)
	}
	endIndex := endMatched[0]

	content := inputText[contentStart : contentStart+endIndex]

	// Trim leading and trailing whitespace
	content = strings.TrimSpace(content)

	return content, nil
}
