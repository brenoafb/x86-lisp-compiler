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
	code := "(let (r 23) (if (zero? 1) 0 r))"
	err = c.Compile(code)

	if err != nil {
		panic(err)
	}
}
