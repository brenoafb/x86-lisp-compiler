package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/brenoafb/tinycompiler/pkg/parser"
	pp "github.com/brenoafb/tinycompiler/pkg/preprocess"
)

var (
	input  = flag.String("i", "", "input file")
	output = flag.String("o", "", "file to write assembly output to")
)

func main() {
	flag.Parse()

	if *input == "" {
		panic("please provide an input file")
	}

	name, _ := strings.CutSuffix(*input, ".lisp")

	content, err := os.ReadFile(*input)
	if err != nil {
		panic("error opening file")
	}

	code := string(content)

	tokens, err := parser.Tokenize(code)

	if err != nil {
		panic(fmt.Errorf("tokenizer error: %w", err))
	}

	es, err := parser.Parse(tokens)

	if err != nil {
		panic(fmt.Errorf("parser error: %w", err))
	}

	e, err := pp.Preprocess(es, name)

	if err != nil {
		panic(fmt.Errorf("error processing expression: %w", err))
	}

	f := os.Stdout

	if *output != "" {
		f, err = os.Create(*output)

		if err != nil {
			panic(fmt.Errorf("cannot open output file: %w", err))
		}

	}

	_, err = f.WriteString(e.String())

	if err != nil {
		panic(fmt.Errorf("error writing file: %w", err))
	}
}
