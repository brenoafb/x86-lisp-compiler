package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/brenoafb/tinycompiler/pkg/compiler"
	"github.com/brenoafb/tinycompiler/pkg/parser"
	pp "github.com/brenoafb/tinycompiler/pkg/preprocess"
)

var (
	input  = flag.String("i", "", "input file")
	output = flag.String("o", "output.s", "file to write assembly output to")
	nopp   = flag.Bool("np", false, "don't pre-process input")
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

	if !*nopp {
		e, err = pp.Preprocess(e)
		if err != nil {
			panic(fmt.Errorf("preprocessor error: %w", err))
		}
	}

	f, err := os.Create(*output)

	if err != nil {
		panic("cannot open output file")
	}

	c := compiler.NewCompiler(f)
	err = c.Compile(e)

	if err != nil {
		panic(err)
	}
}
