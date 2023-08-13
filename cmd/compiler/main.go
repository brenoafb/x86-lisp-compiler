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
	code := `
(labels ((f0 (code (x) (+ x 7))))
  (let (x 13) (labelcall f0 x)))
`
	err = c.Compile(code)

	if err != nil {
		panic(err)
	}
}
