package compiler

import (
	"bytes"
	"fmt"

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
	c := NewCompiler(w)

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
	c := NewCompiler(w)

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

func TestNullP(t *testing.T) {
	code := "(null? ())"
	expected := `    .text
    .globl  scheme_entry
    .p2align    2
scheme_entry:
movl $0x2f, %eax
cmpl $0x2f, %eax
movl $0, %eax
sete %al
sall $7, %eax
orl $0x1f, %eax
ret
`

	w := &bytes.Buffer{}
	c := NewCompiler(w)

	err := c.Compile(code)

	if err != nil {
		t.Errorf("error compiling program: %s", err)
	}

	fmt.Println("RESULT")
	fmt.Println(w.String())

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
orl $0x1f, %ecx
ret
`

	w := &bytes.Buffer{}
	c := NewCompiler(w)

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

func TestAdd(t *testing.T) {
	code := "(+ 13 87)"
	expected := `    .text
    .globl  scheme_entry
    .p2align    2
scheme_entry:
movl $348, %eax
movl %eax, -8(%rsp)
movl $52, %eax
addl -8(%rsp), %eax
ret
`

	w := &bytes.Buffer{}
	c := NewCompiler(w)

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
