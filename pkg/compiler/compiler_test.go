package compiler

import (
	"bytes"

	// "fmt"
	"testing"
)

func TestIntegerExpr(t *testing.T) {
	code := "42"
	expected := `    .text
    .globl  scheme_entry
    .p2align    2
scheme_entry:
movl $168, %eax
ret
`

	w := &bytes.Buffer{}
	c := &Compiler{w}

	err := c.Compile(code)

	if err != nil {
		t.Errorf("error compiling program: %s", err)
	}

	// fmt.Println("RESULT")
	// fmt.Println(w.String())

	if w.String() != expected {
		t.Errorf("emmited code did not match expected output")
	}
}

func TestAdd1(t *testing.T) {
	code := "(add1 41)"
	expected := `    .text
    .globl  scheme_entry
    .p2align    2
scheme_entry:
movl $164, %eax
addl $4, %eax
ret
`

	w := &bytes.Buffer{}
	c := &Compiler{w}

	err := c.Compile(code)

	if err != nil {
		t.Errorf("error compiling program: %s", err)
	}

	// fmt.Println("RESULT")
	// fmt.Println(w.String())

	if w.String() != expected {
		t.Errorf("emmited code did not match expected output")
	}
}

func TestZeroP(t *testing.T) {
	code := "(zero? 41)"
	expected := `    .text
    .globl  scheme_entry
    .p2align    2
scheme_entry:
movl $164, %eax
cmpl $0, %eax
movl $0, %eax
sete %al
sall $7, %eax
orl $0x3f, %ecx
ret
`

	w := &bytes.Buffer{}
	c := &Compiler{w}

	err := c.Compile(code)

	if err != nil {
		t.Errorf("error compiling program: %s", err)
	}

	// fmt.Println("RESULT")
	// fmt.Println(w.String())

	if w.String() != expected {
		t.Errorf("emmited code did not match expected output")
	}
}
