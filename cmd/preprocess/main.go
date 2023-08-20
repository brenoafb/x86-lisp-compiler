package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/brenoafb/tinycompiler/pkg/parser"
	pp "github.com/brenoafb/tinycompiler/pkg/preprocess"
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

	e, err = pp.Preprocess(e)

	if err != nil {
		panic(fmt.Errorf("preprocessor error: %w", err))
	}

	fmt.Println(e.String())
}
