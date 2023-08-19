package parser

import (
	"testing"

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

	require.Equal(t, result, []interface{}{})
}

func TestParseSingletonList(t *testing.T) {
	code := "(hello)"
	expected := []interface{}{
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
	expected := []interface{}{
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
	expected := []interface{}{
		"let",
		[]interface{}{"x", 1},
		"x",
	}

	tokens, err := Tokenize(code)
	require.NoError(t, err)
	result, err := Parse(tokens)
	require.NoError(t, err)

	require.Equal(t, expected, result)
}
