package preprocess

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brenoafb/tinycompiler/pkg/expr"
	"github.com/brenoafb/tinycompiler/pkg/parser"
)

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
			err = gatherFreeVariables(expr, argsMap, freeVars)
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

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) { // Use the code as the descriptor
			tokens, err := parser.Tokenize(tt.code)
			require.NoError(t, err)
			expr, err := parser.Parse(tokens)
			require.NoError(t, err)

			result, err := annotateFreeVariables(expr)
			require.NoError(t, err)

			fmt.Printf("%v\n", result)

			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGatherStrings(t *testing.T) {
	tests := []struct {
		code            string
		expected        expr.E
		gatheredStrings map[string]expr.E
	}{
		{
			code:            "1",
			expected:        expr.N(1),
			gatheredStrings: map[string]expr.E{},
		},
		{
			code:            "(+ 1 2)",
			expected:        expr.L(expr.Id("+"), expr.N(1), expr.N(2)),
			gatheredStrings: map[string]expr.E{},
		},
		{
			code:     `"hello world"`,
			expected: expr.L(expr.Id("string-ref"), expr.Id("s0")),
			gatheredStrings: map[string]expr.E{
				"s0": expr.L(
					expr.Id("string-init"),
					expr.S("hello world"),
				),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) { // Use the code as the descriptor
			tokens, err := parser.Tokenize(tt.code)
			require.NoError(t, err)
			e, err := parser.Parse(tokens)
			require.NoError(t, err)

			strings := make(map[string]expr.E)
			counter := 0

			result, err := gatherStrings(e, &counter, strings)
			require.NoError(t, err)

			require.Equal(t, tt.expected, result)

			require.Equal(t, tt.gatheredStrings, strings)
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

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) { // Use the code as the descriptor
			tokens, err := parser.Tokenize(tt.code)
			require.NoError(t, err)
			e, err := parser.Parse(tokens)
			require.NoError(t, err)

			lambdas := make(map[string]expr.E)
			counter := 0

			result, err := gatherLambdas(e, &counter, lambdas)
			require.NoError(t, err)

			fmt.Printf("%v\n", lambdas)

			require.Equal(t, tt.expected, result)

			require.Equal(t, tt.gatheredLambdas, lambdas)
		})
	}
}

func TestPreprocess(t *testing.T) {
	tests := []struct {
		code     string
		expected expr.E
	}{
		{
			code: "1",
			expected: expr.L(
				expr.Id("_main"),
				expr.L(),
				expr.L(),
				expr.N(1),
			),
		},
		{
			code: "(+ 1 2)",
			expected: expr.L(
				expr.Id("_main"),
				expr.L(),
				expr.L(),
				expr.L(expr.Id("+"), expr.N(1), expr.N(2)),
			),
		},
		{
			code: "(lambda (x) (+ x 1))",
			expected: expr.L(
				expr.Id("_main"),
				expr.L(),
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
				expr.Id("_main"),
				expr.L(),
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
				expr.Id("_main"),
				expr.L(),
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

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) { // Use the code as the descriptor
			tokens, err := parser.Tokenize(tt.code)
			require.NoError(t, err)
			expr, err := parser.Parse(tokens)
			require.NoError(t, err)

			result, err := Preprocess(expr)
			require.NoError(t, err)

			fmt.Printf("%s\n", result.String())

			require.Equal(t, tt.expected, result)
		})
	}
}
