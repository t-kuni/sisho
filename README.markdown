lang: [EN](README.markdown) | [JA](README.ja.markdown)

![Frame 5](https://github.com/user-attachments/assets/94a975ec-4e90-49fe-81b7-c1d358f76a77)

Sisho is a LLM powered knowledge-driven code generation framework.

# Demo

* [Web Page Generation Demo](https://github.com/t-kuni/sisho-demo/tree/master/1-web-page)
* [CLI App Generation Demo](https://github.com/t-kuni/sisho-demo/tree/master/2-cli-app)
* [API Server Generation Demo](https://github.com/t-kuni/sisho-demo/tree/master/3-api-server)

# Install

```
GOPROXY=direct go install github.com/t-kuni/sisho@master
```

# Usage

```
# Initialize project
sisho init

# Add knowledge to generate code
# Syntax: sisho add [kind] [path]
# Example:
sisho add specifications swagger.yml
sisho add specifications er.mmd
sisho add examples handlers/getUser.go
sisho add implementations handlers/getUser.go
sisho add dependencies go.mod

# Code generation
# Syntax: sisho make [target path1] [target path2] ... 
export ANTHROPIC_API_KEY="xxxx"
sisho make -a handlers/postUser.go handlers/deleteUser.go
```

## Development

```
cp .env.example .env
go run main.go 
```

test

```
go generate ./...
go test ./...
```