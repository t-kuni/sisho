lang: [EN](README.markdown) | [JA](README.ja.markdown)

![Frame 5](https://github.com/user-attachments/assets/94a975ec-4e90-49fe-81b7-c1d358f76a77)

Sisho is a LLM powered knowledge-driven code generation tool.

- **Strengths**:
  - By incorporating domain-specific knowledge (e.g., specifications, related code, sample implementations) into the prompts, it can generate highly accurate code.
  - Domain knowledge can be managed using a `.knowledge.yml` file and automatically applied based on the target being generated.
  - It generates multiple files simultaneously, ensuring consistency across them.
  - It accelerates implementation by horizontally expanding based on existing code.
  - It can also generate test code based on the specifications.

- **Concept**:
  - Rather than being fully autonomous, the goal is controlled and precise code generation that avoids turning the code into a "black box."
  - It aims to quickly generate an 80% complete initial implementation, particularly for those capable of designing the software's overall architecture.


# Demo

* [Web Page Generation Demo](https://github.com/t-kuni/sisho-demo/tree/master/1-web-page)
* [CLI App Generation Demo](https://github.com/t-kuni/sisho-demo/tree/master/2-cli-app)
* [API Server Generation Demo](https://github.com/t-kuni/sisho-demo/tree/master/3-api-server)

# Install

```
go install github.com/t-kuni/sisho@latest
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