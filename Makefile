CC=zig cc -target x86-linux-musl
OPTS=-g

all: main

expr: pkg/expr/*.go
	go build ./pkg/expr/

preprocess: expr cmd/preprocess/*.go pkg/preprocess/*.go
	go install ./cmd/preprocess/

compiler: expr cmd/compiler/*.go pkg/compiler/*.go
	go install ./cmd/compiler/

%.s: %.lisp compiler
	compiler -i $< -o $@ -np

main: runtime.c lisp_entry.s
	$(CC) $(OPTS) runtime.c lisp_entry.s -o main

.PHONY: test
test: 
	go test ./pkg/compiler ./pkg/parser ./pkg/preprocess

.PHONY: run 
run: main
	qemu-i386-static main

