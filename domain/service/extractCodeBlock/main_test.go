package extractCodeBlock

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractCodeBlock(t *testing.T) {
	service := NewCodeBlockExtractService()

	confusingText := "(?s)(\\n|^)<!-- CODE_BLOCK_BEGIN -->```example.txt\n(.*)```.?<!-- CODE_BLOCK_END -->(\\n|$)"
	tests := []struct {
		name      string
		input     string
		filePath  string
		expected  string
		expectErr bool
	}{
		{
			name: "正常パターン",
			input: `<!-- CODE_BLOCK_BEGIN -->` + "```" + `test/file.go
CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->`,
			filePath:  "test/file.go",
			expected:  `CONTENT`,
			expectErr: false,
		},
		{
			name: "前後にテキストがあるパターン",
			input: `Some text before
<!-- CODE_BLOCK_BEGIN -->` + "```" + `test/file.go
CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->
Some text after`,
			filePath:  "test/file.go",
			expected:  `CONTENT`,
			expectErr: false,
		},
		{
			name: "関係ないファイルのコードブロックしかないパターン",
			input: `Some text
<!-- CODE_BLOCK_BEGIN -->` + "```" + `other/file.go
CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->`,
			filePath:  "test/file.go",
			expected:  "",
			expectErr: true,
		},
		{
			name: "複数のコードブロックが存在するパターン1",
			input: `<!-- CODE_BLOCK_BEGIN -->` + "```" + `other/file.go
CONTENT1
` + "```" + `<!-- CODE_BLOCK_END -->
<!-- CODE_BLOCK_BEGIN -->` + "```" + `test/file.go
CONTENT2
` + "```" + `<!-- CODE_BLOCK_END -->`,
			filePath:  "test/file.go",
			expected:  "CONTENT2",
			expectErr: false,
		},
		{
			name: "複数のコードブロックが存在するパターン2",
			input: `<!-- CODE_BLOCK_BEGIN -->` + "```" + `test/file.go
CONTENT1
` + "```" + `<!-- CODE_BLOCK_END -->
<!-- CODE_BLOCK_BEGIN -->` + "```" + `other/file.go
CONTENT2
` + "```" + `<!-- CODE_BLOCK_END -->`,
			filePath:  "test/file.go",
			expected:  "CONTENT1",
			expectErr: false,
		},
		{
			name: "開始タグが行頭にない場合は対象とならないこと",
			input: `DUMMY<!-- CODE_BLOCK_BEGIN -->` + "```" + `test/file.go
CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->`,
			filePath:  "test/file.go",
			expected:  ``,
			expectErr: true,
		},
		{
			name: "終了タグが行頭にない場合は対象とならないこと",
			input: `<!-- CODE_BLOCK_BEGIN -->` + "```" + `test/file.go
CONTENT
DUMMY` + "```" + `<!-- CODE_BLOCK_END -->`,
			filePath:  "test/file.go",
			expected:  ``,
			expectErr: true,
		},
		{
			name: "ファイルパスに正規表現のメタ文字が含まれるパターン",
			input: `<!-- CODE_BLOCK_BEGIN -->` + "```" + `test/.*+?file.go
CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->`,
			filePath:  "test/.*+?file.go",
			expected:  `CONTENT`,
			expectErr: false,
		},
		{
			name: "コーロブロック内が空のパターン",
			input: `<!-- CODE_BLOCK_BEGIN -->` + "```" + `test/file.go
` + "```" + `<!-- CODE_BLOCK_END -->`,
			filePath:  "test/file.go",
			expected:  "",
			expectErr: false,
		},
		{
			name: "終了タグに改行が含まれるパターン（ChatGPTで発生する）",
			input: `<!-- CODE_BLOCK_BEGIN -->` + "```" + `test/file.go
CONTENT
` + "```\n" + `<!-- CODE_BLOCK_END -->`,
			filePath:  "test/file.go",
			expected:  "CONTENT",
			expectErr: false,
		},
		{
			name: "コードブロック内に紛らわしいテキストが含まれるパターン", // Sisho自身を生成するケースを想定（extractCodeBlock自身を生成するようなケース）
			input: `<!-- CODE_BLOCK_BEGIN -->` + "```" + `test/file.go
CONTENT ` + confusingText + `
` + "```" + `<!-- CODE_BLOCK_END -->
`,
			filePath:  "test/file.go",
			expected:  `CONTENT ` + confusingText,
			expectErr: false,
		},
		{
			name:      "コードブロックがないパターン",
			input:     `Some text`,
			filePath:  "test/file.go",
			expected:  "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.ExtractCodeBlock(tt.input, tt.filePath)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
