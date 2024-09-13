package main

import (
	_ "embed"
	"github.com/joho/godotenv"
	"github.com/t-kuni/sisho/cmd"
	"path/filepath"
	"runtime"
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
