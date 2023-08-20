package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/brenoafb/tinycompiler/pkg/parser"
)

var (
	input = flag.String("i", "", "input file")
)

func main() {
	flag.Parse()

	if *input == "" {
		panic("please provide an input file")
	}

	content, err := os.ReadFile(*input)
	if err != nil {
		panic("error opening file")
	}

	code := string(content)

	tokens, err := parser.Tokenize(code)

	if err != nil {
		panic(fmt.Errorf("tokenizer error: %w", err))
	}

	e, err := parser.Parse(tokens)

	if err != nil {
		panic(fmt.Errorf("parser error: %w", err))
	}

	fmt.Println(e.String())
}
