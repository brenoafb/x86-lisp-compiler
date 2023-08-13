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
(labels ((c0 (code (x) () (+ x 21))))
  (funcall (closure c0) 3))
`
	err = c.Compile(code)

	if err != nil {
		panic(err)
	}
}
