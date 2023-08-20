CC=zig cc -target x86-linux-musl
OPTS=-g

all: main

compiler: cmd/compiler/*.go pkg/compiler/*.go
	go build -o compiler ./cmd/compiler/

output.s: compiler
	./compiler -o output.s

main: runtime.c output.s
	$(CC) $(OPTS) runtime.c output.s -o main

.PHONY: test
test: 
	go test ./pkg/compiler ./pkg/parser ./pkg/preprocess

.PHONY: run 
run: main
	qemu-i386-static main

