
# Commands for notes
default:
  @just --list
# Build notes with Go
build:
  go build ./...

# Run tests for notes with Go
test:
  go clean -testcache
  go test ./...