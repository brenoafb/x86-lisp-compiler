package parser

import (
	"testing"

	"github.com/brenoafb/tinycompiler/pkg/expr"
	"github.com/stretchr/testify/require"
)

func TestTokenize(t *testing.T) {
	tcs := []struct {
		code     string
		expected []Token
	}{
		{
			code:     "",
			expected: []Token{},
		},
		{
			code: "()",
			expected: []Token{
				lparen(),
				rparen(),
			},
		},
		{
			code: "(zero? 0)",
			expected: []Token{
				lparen(),
				ident("zero?"),
				number(0),
				rparen(),
			},
		},
		{
			code: "(hello world)",
			expected: []Token{
				lparen(),
				ident("hello"),
				ident("world"),
				rparen(),
			},
		},
		{
			code: "(hello       world)",
			expected: []Token{
				lparen(),
				ident("hello"),
				ident("world"),
				rparen(),
			},
		},
		{
			code: "(hello (world))",
			expected: []Token{
				lparen(),
				ident("hello"),
				lparen(),
				ident("world"),
				rparen(),
				rparen(),
			},
		},
		{
			code: `(hello 
 (world))`,
			expected: []Token{
				lparen(),
				ident("hello"),
				lparen(),
				ident("world"),
				rparen(),
				rparen(),
			},
		},
		{
			code: `(hello 
          (world)
     )`,
			expected: []Token{
				lparen(),
				ident("hello"),
				lparen(),
				ident("world"),
				rparen(),
				rparen(),
			},
		},
		{
			code: `"hello world"`,
			expected: []Token{
				str("hello world"),
			},
		},
		{
			code: `(  "hello" ("world" ))`,
			expected: []Token{
				lparen(),
				str("hello"),
				lparen(),
				str("world"),
				rparen(),
				rparen(),
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.code, func(t *testing.T) {
			ts, err := Tokenize(tc.code)
			require.NoError(t, err)
			require.Equal(t, tc.expected, ts.tokens)
		})
	}
}

func TestParseEmptyList(t *testing.T) {
	code := "()"
	tokens, err := Tokenize(code)
	require.NoError(t, err)
	result, err := parseExpr(tokens)
	require.NoError(t, err)

	require.Equal(t, result, expr.Nil())
}

func TestParseSingletonList(t *testing.T) {
	code := "(hello)"
	expected := expr.L(expr.Id("hello"))
	tokens, err := Tokenize(code)
	require.NoError(t, err)
	result, err := parseExpr(tokens)
	require.NoError(t, err)

	require.Equal(t, expected, result)
}

func TestParseFlatList(t *testing.T) {
	code := "(hello world)"
	expected := expr.L(expr.Id("hello"), expr.Id("world"))

	tokens, err := Tokenize(code)
	require.NoError(t, err)
	result, err := parseExpr(tokens)
	require.NoError(t, err)

	require.Equal(t, expected, result)
}

func TestParseLet(t *testing.T) {
	code := "(let (x 1) x)"
	expected := expr.L(
		expr.Id("let"),
		expr.L(expr.Id("x"), expr.N(1)),
		expr.Id("x"),
	)

	tokens, err := Tokenize(code)
	require.NoError(t, err)
	result, err := parseExpr(tokens)
	require.NoError(t, err)

	require.Equal(t, expected, result)
}

func TestParseMultipleExprs(t *testing.T) {
	code := "(+ x 1) (+ x 2)"
	expected := []expr.E{
		expr.L(
			expr.Id("+"),
			expr.Id("x"),
			expr.N(1),
		),
		expr.L(
			expr.Id("+"),
			expr.Id("x"),
			expr.N(2),
		),
	}

	tokens, err := Tokenize(code)
	require.NoError(t, err)
	result, err := Parse(tokens)
	require.NoError(t, err)

	require.Equal(t, expected, result)
}
