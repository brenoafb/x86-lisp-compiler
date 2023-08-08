CC=zig cc -target x86_64-linux-musl

all: main

output.s: cmd/compiler/main.go pkg/compiler/compiler.go
	go run ./cmd/compiler/main.go -- -o output.s

main: runtime.c output.s
	$(CC) $(OPTS) runtime.c output.s -o main

.PHONY: test
test: 
	go test ./pkg/compiler ./pkg/parser

.PHONY: run 
run: main
	BLINK_PREFIX=/tmp/blink blink -m main