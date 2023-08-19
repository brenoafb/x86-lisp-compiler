package compiler

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brenoafb/tinycompiler/pkg/parser"
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

func TestGatherFreeVariables(t *testing.T) {
	tests := []struct {
		code             string
		expectedFreeVars []string
		args             []string
	}{
		{
			code:             "(a b c x y z a b c)",
			expectedFreeVars: []string{"x", "y", "z"},
			args:             []string{"a", "b", "c"},
		},
		{
			code:             "(a (b c) x y (z a) b c)",
			expectedFreeVars: []string{"x", "y", "z"},
			args:             []string{"a", "b", "c"},
		},
		{
			code:             "((a b) c (x y z) a b c)",
			expectedFreeVars: []string{"x", "y", "z"},
			args:             []string{"a", "b", "c"},
		},
		{
			code:             "(a (b (c (x y) z) a) b c)",
			expectedFreeVars: []string{"x", "y", "z"},
			args:             []string{"a", "b", "c"},
		},
	}

	w := &bytes.Buffer{}
	c := NewCompiler(w)

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) { // Use the code as the descriptor
			// Convert the slice of args into a map
			argsMap := make(map[string]struct{})
			for _, arg := range tt.args {
				argsMap[arg] = struct{}{}
			}

			tokens := parser.Tokenize(tt.code)
			expr, err := parser.Parse(tokens)
			require.NoError(t, err)

			freeVars := map[string]struct{}{}
			err = c.gatherFreeVariables(expr, argsMap, freeVars)
			require.NoError(t, err)

			for _, key := range tt.expectedFreeVars {
				require.Contains(t, freeVars, key, "Free vars should contain "+key)
			}
		})
	}
}

func TestAnnotateFreeVariables(t *testing.T) {
	tests := []struct {
		code     string
		expected interface{}
	}{
		{
			code:     "1",
			expected: 1,
		},
		{
			code:     "a",
			expected: "a",
		},
		{
			code:     "()",
			expected: []interface{}{},
		},
		{
			code:     "(+ 1 2)",
			expected: []interface{}{"+", 1, 2},
		},
		{
			code: "(lambda (x) (f x 1))",
			expected: []interface{}{
				"lambda",
				[]interface{}{"x"},
				[]interface{}{"f"},
				[]interface{}{"f", "x", 1},
			},
		},
		{
			code: "(lambda (x) (+ x 1))",
			expected: []interface{}{
				"lambda",
				[]interface{}{"x"},
				[]interface{}{},
				[]interface{}{"+", "x", 1},
			},
		},
		{
			code: "(lambda (y) (lambda () (+ x y)))",
			expected: []interface{}{
				"lambda",
				[]interface{}{"y"},
				[]interface{}{"x"},
				[]interface{}{
					"lambda",
					[]interface{}{},
					[]interface{}{"x", "y"},
					[]interface{}{"+", "x", "y"},
				},
			},
		},
	}

	w := &bytes.Buffer{}
	c := NewCompiler(w)

	builtinsMap := make(map[string]struct{})

	for k := range builtins {
		builtinsMap[k] = struct{}{}
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) { // Use the code as the descriptor
			tokens := parser.Tokenize(tt.code)
			expr, err := parser.Parse(tokens)
			require.NoError(t, err)

			result, err := c.annotateFreeVariables(expr)
			require.NoError(t, err)

			fmt.Printf("%v\n", result)

			require.Equal(t, tt.expected, result)
		})
	}
}
