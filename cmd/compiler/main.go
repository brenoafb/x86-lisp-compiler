package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/brenoafb/tinycompiler/pkg/compiler"
	"github.com/brenoafb/tinycompiler/pkg/expr"
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

	var e expr.E
	if *nopp {
		if len(es) != 1 {
			panic("preprocessed expressions should have a single element")
		}
		e = es[0]
	} else {
		e, err = pp.Preprocess(es, name)
		if err != nil {
			panic(fmt.Errorf("preprocessor error: %w", err))
		}
	}

	f, err := os.Create(*output)

	if err != nil {
		panic("cannot open output file")
	}

	defer f.Close()

	c := compiler.NewCompiler(f)
	err = c.Compile(e)

	if err != nil {
		panic(err)
	}
}
