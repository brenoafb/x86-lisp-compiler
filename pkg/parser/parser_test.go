package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/brenoafb/tinycompiler/pkg/ast"
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
	result, err := Parse(tokens)
	require.NoError(t, err)

	require.Equal(t, result, []ast.Expr{})
}

func TestParseSingletonList(t *testing.T) {
	code := "(hello)"
	expected := []ast.Expr{
		"hello",
	}
	tokens, err := Tokenize(code)
	require.NoError(t, err)
	result, err := Parse(tokens)
	require.NoError(t, err)

	require.Equal(t, expected, result)
}

func TestParseFlatList(t *testing.T) {
	code := "(hello world)"
	expected := []ast.Expr{
		"hello",
		"world",
	}
	tokens, err := Tokenize(code)
	require.NoError(t, err)
	result, err := Parse(tokens)
	require.NoError(t, err)

	require.Equal(t, expected, result)
}

func TestParseLet(t *testing.T) {
	code := "(let (x 1) x)"
	expected := []ast.Expr{
		"let",
		[]ast.Expr{"x", 1},
		"x",
	}

	tokens, err := Tokenize(code)
	require.NoError(t, err)
	result, err := Parse(tokens)
	require.NoError(t, err)

	require.Equal(t, expected, result)
}
