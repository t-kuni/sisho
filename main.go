package main

import (
	_ "embed"
	"fmt"
	"github.com/t-kuni/llm-coding-example/llm/chat/claude"
	"github.com/t-kuni/llm-coding-example/llm/cmd"
	"github.com/t-kuni/llm-coding-example/llm/prompts/begin"
	"github.com/t-kuni/llm-coding-example/llm/prompts/conclusion"
	"github.com/t-kuni/llm-coding-example/llm/prompts/readDoc"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	_, currentFilePath, _, _ := runtime.Caller(0)
	dir := filepath.Dir(currentFilePath)
	envPath := filepath.Join(dir, ".env")
	godotenv.Load(envPath)

	err := cmd.NewRootCommand().CobraCommand.Execute()
	if err != nil {
		panic(err)
	}
}

func mainOld() {
	if len(os.Args) < 2 {
		panic("引数が不足しています")
	}

	err := godotenv.Load("llm/.env")
	if err != nil {
		panic(err)
	}

	tree, err := makeTree(".")
	if err != nil {
		panic(err)
	}

	requirements, err := readFile("llm/requirements.txt")
	if err != nil {
		panic(err)
	}

	chat := claude.ClaudeChat{}
	var answer begin.Answer
	{
		prompt, err := begin.BuildPrompt(begin.PromptParam{
			Tree:         tree,
			Requirements: requirements,
		})
		if err != nil {
			panic(err)
		}

		answerText, err := chat.Send(prompt)
		if err != nil {
			panic(err)
		}

		err = answer.Parse(answerText)
		if err != nil {
			panic(err)
		}
	}

	_, err = try(3, func() (bool, error) {
		if answer.Type != "read" {
			return false, nil
		}

		var param readDoc.PromptParam
		for _, path := range answer.Read.Paths {
			fmt.Printf("Read: %s\n", path)

			content, err := readFile(path)
			if err != nil {
				panic(err)
			}

			param.Documents = append(param.Documents, readDoc.PromptParamDoc{
				Path:    path,
				Content: content,
			})
		}

		prompt, err := readDoc.BuildPrompt(param)
		if err != nil {
			panic(err)
		}

		answerJson, err := chat.Send(prompt)
		if err != nil {
			panic(err)
		}

		err = answer.Parse(answerJson)
		if err != nil {
			panic(err)
		}

		return true, nil
	})
	if err != nil {
		panic(err)
	}

	if answer.Type != "ok" {
		panic("結論フェーズに移行できませんでした")
		return
	}

	{
		prompt, err := conclusion.BuildPrompt(conclusion.PromptParam{})
		if err != nil {
			panic(err)
		}

		answerJson, err := chat.Send(prompt)
		if err != nil {
			panic(err)
		}

		fmt.Println(answerJson)
	}
}

func readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func makeTree(root string) (string, error) {
	lines := []string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 隠しフォルダをスキップ
		if info.Name() != "." && info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// インデントを付ける
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		depth := strings.Count(relativePath, string(os.PathSeparator))

		// ディレクトリかどうかを判定
		prefix := ""
		if info.IsDir() {
			if depth == 0 {
				// 最上位のディレクトリには ./ を付ける
				prefix = "./"
			} else {
				// サブディレクトリには / を付ける
				prefix = "/"
			}
		}

		// インデント付きでファイルやディレクトリを出力
		line := fmt.Sprintf("%s%s%s\n", strings.Repeat("  ", depth), prefix, info.Name())
		lines = append(lines, line)

		return nil
	})
	if err != nil {
		return "", err
	}
	return strings.Join(lines, ""), nil
}

func try(n int, closure func() (bool, error)) (bool, error) {
	for i := 0; i < n; i++ {
		success, err := closure()
		if err != nil {
			return false, err
		}
		if !success {
			return success, nil
		}
	}
	return true, nil
}
