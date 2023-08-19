package compiler

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brenoafb/tinycompiler/pkg/expr"
	"github.com/brenoafb/tinycompiler/pkg/parser"
)

func TestCompileExpr(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{
			code:     "42",
			expected: "movl $168, %eax\n",
		},
		{
			code: "(add1 42)",
			expected: `movl $168, %eax
addl $4, %eax
`,
		},
		{
			code: "(null? ())",
			expected: `movl $0x2f, %eax
cmpl $0x2f, %eax
movl $0, %eax
sete %al
sall $7, %eax
orl $0x1f, %eax
`,
		},
		{
			code: "(zero? 41)",
			expected: `movl $164, %eax
cmpl $0, %eax
movl $0, %eax
sete %al
sall $7, %eax
orl $0x1f, %eax
`,
		},
		{
			code: "(+ 13 87)",
			expected: `movl $348, %eax
movl %eax, -4(%esp)
movl $52, %eax
addl -4(%esp), %eax
`,
		},
		{
			code: "(let (x 1) x)",
			expected: `movl $4, %eax
movl %eax, -4(%esp)
movl -4(%esp), %eax
`,
		},
		{
			code: "(let (x 1) (y 2) (+ x y))",
			expected: `movl $4, %eax
movl %eax, -4(%esp)
movl $8, %eax
movl %eax, -8(%esp)
movl -8(%esp), %eax
movl %eax, -12(%esp)
movl -4(%esp), %eax
addl -12(%esp), %eax
`,
		},
		{
			code: "(if (zero? 1) 0 1)",
			expected: `movl $4, %eax
cmpl $0, %eax
movl $0, %eax
sete %al
sall $7, %eax
orl $0x1f, %eax
cmpl $0x1f, %eax
je L0
movl $0, %eax
jmp L1
L0:
movl $4, %eax
L1:
`,
		},
		{
			code: "(cons 1 2)",
			expected: `movl $4, %eax
movl %eax, 0(%esi)
movl $8, %eax
movl %eax, 4(%esi)
movl %esi, %eax
orl $1, %eax
addl $8, %esi
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			w := &bytes.Buffer{}
			c := NewCompiler(w)

			tokens, err := parser.Tokenize(tt.code)
			require.NoError(t, err)
			expr, err := parser.Parse(tokens)
			require.NoError(t, err)

			err = c.compileExpr(expr)
			require.NoError(t, err)
			require.Equal(t, tt.expected, w.String())
		})
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

			tokens, err := parser.Tokenize(tt.code)
			require.NoError(t, err)
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
		expected expr.E
	}{
		{
			code:     "1",
			expected: expr.N(1),
		},
		{
			code:     "a",
			expected: expr.Id("a"),
		},
		{
			code:     "()",
			expected: expr.Nil(),
		},
		{
			code:     "(+ 1 2)",
			expected: expr.L(expr.Id("+"), expr.N(1), expr.N(2)),
		},
		{
			code: "(lambda (x) (f x 1))",
			expected: expr.L(
				expr.Id("lambda"),
				expr.L(expr.Id("x")),
				expr.L(expr.Id("f")),
				expr.L(expr.Id("f"), expr.Id("x"), expr.N(1)),
			),
		},
		{
			code: "(lambda (x) (+ x 1))",
			expected: expr.L(
				expr.Id("lambda"),
				expr.L(expr.Id("x")),
				expr.L(),
				expr.L(expr.Id("+"), expr.Id("x"), expr.N(1)),
			),
		},
		{
			code: "(lambda (y) (lambda () (+ x y)))",
			expected: expr.L(
				expr.Id("lambda"),
				expr.L(expr.Id("y")),
				expr.L(expr.Id("x")),
				expr.L(
					expr.Id("lambda"),
					expr.L(),
					expr.L(expr.Id("x"), expr.Id("y")),
					expr.L(expr.Id("+"), expr.Id("x"), expr.Id("y")),
				),
			),
		},
	}

	w := &bytes.Buffer{}
	c := NewCompiler(w)

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) { // Use the code as the descriptor
			tokens, err := parser.Tokenize(tt.code)
			require.NoError(t, err)
			expr, err := parser.Parse(tokens)
			require.NoError(t, err)

			result, err := c.annotateFreeVariables(expr)
			require.NoError(t, err)

			fmt.Printf("%v\n", result)

			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGatherLambdas(t *testing.T) {
	tests := []struct {
		code            string
		expected        expr.E
		gatheredLambdas map[string]expr.E
	}{
		{
			code:            "1",
			expected:        expr.N(1),
			gatheredLambdas: map[string]expr.E{},
		},
		{
			code:            "(+ 1 2)",
			expected:        expr.L(expr.Id("+"), expr.N(1), expr.N(2)),
			gatheredLambdas: map[string]expr.E{},
		},
		{
			code:     "(lambda (x) () (+ x 1))",
			expected: expr.L(expr.Id("closure"), expr.Id("f0")),
			gatheredLambdas: map[string]expr.E{
				"f0": expr.L(
					expr.Id("code"),
					expr.L(expr.Id("x")), // args
					expr.L(),             // free vars
					expr.L(expr.Id("+"), expr.Id("x"), expr.N(1)), // body
				),
			},
		},
		{
			code: "((lambda (x) () (+ x 1)) 1)",
			expected: expr.L(
				expr.L(expr.Id("closure"), expr.Id("f0")),
				expr.N(1),
			),
			gatheredLambdas: map[string]expr.E{
				"f0": expr.L(
					expr.Id("code"),
					expr.L(expr.Id("x")), // args
					expr.L(),             // free vars
					expr.L(expr.Id("+"), expr.Id("x"), expr.N(1)), // body
				),
			},
		},
		{
			code: "(lambda (y) (x) (lambda () (x y) (+ x y)))",
			expected: expr.L(
				expr.Id("closure"),
				expr.Id("f1"),
				expr.Id("x"),
			),
			gatheredLambdas: map[string]expr.E{
				"f0": expr.L(
					expr.Id("code"),
					expr.L(),                           // args
					expr.L(expr.Id("x"), expr.Id("y")), // free vars
					expr.L(expr.Id("+"), expr.Id("x"), expr.Id("y")), // body
				),
				"f1": expr.L(
					expr.Id("code"),
					expr.L(expr.Id("y")), // args
					expr.L(expr.Id("x")), // free vars
					expr.L(expr.Id("closure"), expr.Id("f0"), expr.Id("x"), expr.Id("y")), // body
				),
			},
		},
	}

	w := &bytes.Buffer{}
	c := NewCompiler(w)

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) { // Use the code as the descriptor
			tokens, err := parser.Tokenize(tt.code)
			require.NoError(t, err)
			e, err := parser.Parse(tokens)
			require.NoError(t, err)

			lambdas := make(map[string]expr.E)
			counter := 0

			result, err := c.gatherLambdas(e, &counter, lambdas)
			require.NoError(t, err)

			fmt.Printf("%v\n", lambdas)

			require.Equal(t, tt.expected, result)

			require.Equal(t, tt.gatheredLambdas, lambdas)
		})
	}
}

func TestPreProcess(t *testing.T) {
	tests := []struct {
		code     string
		expected expr.E
	}{
		{
			code: "1",
			expected: expr.L(
				expr.Id("labels"),
				expr.L(),
				expr.N(1),
			),
		},
		{
			code: "(+ 1 2)",
			expected: expr.L(
				expr.Id("labels"),
				expr.L(),
				expr.L(expr.Id("+"), expr.N(1), expr.N(2)),
			),
		},
		{
			code: "(lambda (x) (+ x 1))",
			expected: expr.L(
				expr.Id("labels"),
				expr.L(
					expr.L(
						expr.Id("f0"),
						expr.L(
							expr.Id("code"),
							expr.L(expr.Id("x")), // args
							expr.L(),             // free vars
							expr.L(expr.Id("+"), expr.Id("x"), expr.N(1)), // body
						),
					),
				),
				expr.L(expr.Id("closure"), expr.Id("f0")),
			),
		},
		{
			code: "((lambda (x) (+ x 1)) 1)",
			expected: expr.L(
				expr.Id("labels"),
				expr.L(
					expr.L(
						expr.Id("f0"),
						expr.L(
							expr.Id("code"),
							expr.L(expr.Id("x")), // args
							expr.L(),             // free vars
							expr.L(expr.Id("+"), expr.Id("x"), expr.N(1)), // body
						),
					),
				),
				expr.L(
					expr.L(expr.Id("closure"), expr.Id("f0")),
					expr.N(1),
				),
			),
		},
		{
			code: "(lambda (y) (lambda () (+ x y)))",
			expected: expr.L(
				expr.Id("labels"),
				expr.L(
					expr.L(
						expr.Id("f0"),
						expr.L(
							expr.Id("code"),
							expr.L(),                           // args
							expr.L(expr.Id("x"), expr.Id("y")), // free vars
							expr.L(expr.Id("+"), expr.Id("x"), expr.Id("y")), // body
						),
					),
					expr.L(
						expr.Id("f1"),
						expr.L(
							expr.Id("code"),
							expr.L(expr.Id("y")), // args
							expr.L(expr.Id("x")), // free vars
							expr.L(expr.Id("closure"), expr.Id("f0"), expr.Id("x"), expr.Id("y")), // body
						),
					),
				),
				expr.L(expr.Id("closure"), expr.Id("f1"), expr.Id("x")),
			),
		},
	}

	w := &bytes.Buffer{}
	c := NewCompiler(w)

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) { // Use the code as the descriptor
			tokens, err := parser.Tokenize(tt.code)
			require.NoError(t, err)
			expr, err := parser.Parse(tokens)
			require.NoError(t, err)

			result, err := c.preprocess(expr)
			require.NoError(t, err)

			fmt.Printf("%s\n", result.String())

			require.Equal(t, tt.expected, result)
		})
	}
}
