package main

import (
	_ "embed"
	"github.com/joho/godotenv"
	"github.com/t-kuni/sisho/cmd"
)

func main() {
	godotenv.Load(".env")

	err := cmd.NewRootCommand().CobraCommand.Execute()
	if err != nil {
		panic(err)
	}
}
