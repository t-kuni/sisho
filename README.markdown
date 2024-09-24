lang: [EN](README.markdown) | [JA](README.ja.markdown)

![Frame 5](https://github.com/user-attachments/assets/94a975ec-4e90-49fe-81b7-c1d358f76a77)

Sisho is a LLM powered knowledge-driven code generation tool.

- Strengths
  - High-precision code generation by including domain knowledge<font color="red">*</font> in the prompt
  - Domain knowledge<font color="red">*</font> can be managed with the `.knowledge.yml` file and automatically used according to the generation target
  - Maintain consistency between files by generating multiple files simultaneously
  - Speed up horizontal deployment based on existing implementations
  - Generate test code based on specifications

<span style="coloe: red">*</span> Specifications, related code, sample implementations, etc.

- Concept
  - Aiming for controllable and high-precision generation instead of fully automated autonomous type (code does not become a black box)
  - Aiming to quickly generate an initial implementation of about 80 points for those who can create a grand design of software


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