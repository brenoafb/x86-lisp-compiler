CC=zig cc -target x86-linux-musl
OPTS=-g

all: main

expr: pkg/expr/*.go
	go build ./pkg/expr/

.PHONY: test
preprocess: expr cmd/preprocess/*.go pkg/preprocess/*.go
	go install ./cmd/preprocess/

.PHONY: test
compiler: expr cmd/compiler/*.go pkg/compiler/*.go
	go install ./cmd/compiler/

%.pp.lisp: %.lisp preprocess
	preprocess -i $< -o $@

%.s: %.pp.lisp compiler
	compiler -i $< -o $@ -np

main: runtime.c lisp_entry.s stdlib.s
	$(CC) $(OPTS) runtime.c *.s -o main

.PHONY: test
test: 
	go test ./pkg/compiler ./pkg/parser ./pkg/preprocess

.PHONY: run 
run: main
	qemu-i386-static main

