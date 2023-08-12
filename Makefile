CC=zig cc -target x86-linux-musl
OPTS=-g

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
	qemu-i386-static main

