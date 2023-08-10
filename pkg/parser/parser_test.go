package parser

import (
	"testing"
)

func TestTokenize(t *testing.T) {
	tcs := []struct {
		code     string
		expected []string
	}{
		{
			code:     "",
			expected: []string{},
		},
		{
			code: "()",
			expected: []string{
				"(",
				")",
			},
		},
		{
			code: "(hello world)",
			expected: []string{
				"(",
				"hello",
				"world",
				")",
			},
		},
		{
			code: "(hello       world)",
			expected: []string{
				"(",
				"hello",
				"world",
				")",
			},
		},
		{
			code: "(hello (world))",
			expected: []string{
				"(",
				"hello",
				"(",
				"world",
				")",
				")",
			},
		},
	}

	aux := func(input string, expected []string) {
		ts := Tokenize(input)
		tokens := ts.tokens
		if len(expected) != len(tokens) {
			t.Errorf("tokenizer output does not have the expected length")
		}

		for i, token := range tokens {
			if token != expected[i] {
				t.Errorf("bad token at index %d: %s %s", i, token, expected[i])
			}
		}
	}

	for _, tc := range tcs {
		t.Run(tc.code, func(t *testing.T) {
			aux(tc.code, tc.expected)
		})
	}
}

func TestParseAtom(t *testing.T) {
	tcs := []struct {
		code     string
		expected interface{}
	}{
		{
			code:     "42",
			expected: 42,
		},
		{
			code:     "abc",
			expected: "abc",
		},
	}

	aux := func(code string, expected interface{}) {
		tokens := Tokenize(code)
		result, err := Parse(tokens)

		if err != nil {
			t.Errorf("error when parsing input: %s", err)
		}

		if result != expected {
			t.Errorf("parse result is wrong")
		}
	}

	for _, tc := range tcs {
		t.Run(tc.code, func(t *testing.T) {
			aux(tc.code, tc.expected)
		})
	}
}

func TestParseEmptyList(t *testing.T) {
	code := "()"
	tokens := Tokenize(code)
	result, err := Parse(tokens)

	if err != nil {
		t.Errorf("error when parsing input: %s", err)
	}

	switch result.(type) {
	case []interface{}:
		if len(result.([]interface{})) != 0 {
			t.Errorf("parsing empty list")
		}
	default:
		t.Errorf("received value is not a list")
	}

}

func TestParseSingletonList(t *testing.T) {
	code := "(hello)"
	expected := []interface{}{
		"hello",
	}
	tokens := Tokenize(code)
	result, err := Parse(tokens)

	if err != nil {
		t.Errorf("error when parsing input: %s", err)
	}

	switch result.(type) {
	case []interface{}:
		elems := result.([]interface{})
		if len(elems) != len(expected) {
			t.Errorf("element count mismatch when parsing list")
		}

		for i, elem := range elems {
			if elem != expected[i] {
				t.Errorf("wrong element at index %d when parsing list: %s %s", i, elem, expected[i])
			}
		}
	default:
		t.Errorf("received value is not a list")
	}
}


func TestParseFlatList(t *testing.T) {
	code := "(hello world)"
	expected := []interface{}{
		"hello",
		"world",
	}
	tokens := Tokenize(code)
	result, err := Parse(tokens)

	if err != nil {
		t.Errorf("error when parsing input: %s", err)
	}

	switch result.(type) {
	case []interface{}:
		elems := result.([]interface{})
		if len(elems) != len(expected) {
			t.Errorf("element count mismatch when parsing list")
		}

		for i, elem := range elems {
			if elem != expected[i] {
				t.Errorf("wrong element at index %d when parsing list: %s %s", i, elem, expected[i])
			}
		}
	default:
		t.Errorf("received value is not a list")
	}
}

func TestParseLet(t *testing.T) {
	code := "(let (x 1) x)"
	tokens := Tokenize(code)
	result, err := Parse(tokens)

	if err != nil {
		t.Errorf("error when parsing input: %s", err)
	}

	switch result.(type) {
	case []interface{}:
	default:
		t.Errorf("received value is not a list")
	}

	elems := result.([]interface{})
	if len(elems) != 3 {
		t.Errorf("element count mismatch when parsing list")
	}

	if elems[0] != "let" {
		t.Errorf("mismatch on first element")
	}

	switch elems[1].(type) {
		case []interface{}:
		default:
			t.Errorf("second element is not a list")
	}

	second := elems[1].([]interface{})
	if second[0] != "x" {
		t.Errorf("mismatch on first nested element")
	}

	if second[1] != 1 {
		t.Errorf("mismatch on second nested element")
	}

	if elems[2] != "x" {
		t.Errorf("mismatch on third element")
	}
}

