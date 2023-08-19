package main

import (
	"flag"
	"os"

	"github.com/brenoafb/tinycompiler/pkg/compiler"
)

var (
	output = flag.String("o", "output.s", "file to write assembly output to")
)

func main() {
	flag.Parse()

	f, err := os.Create(*output)

	if err != nil {
		panic("cannot open output file")
	}

	c := compiler.NewCompiler(f)
	code := "((lambda (x) (+ x 1)) 1)"
	err = c.Compile(code)

	if err != nil {
		panic(err)
	}
}
