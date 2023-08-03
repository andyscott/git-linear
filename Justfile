build:
  go build -o ./bin/ ./cmd/*

run: build
  ./bin/git-linear