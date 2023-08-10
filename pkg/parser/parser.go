package parser

import (
	"fmt"
	"strconv"
	"strings"
)

func pop(x []string) (string, []string) {
	if len(x) == 0 {
		panic("cannot pop off empty slice")
	}
	return x[0], x[1:]
}

func isAtom(x interface{}) bool {
	switch x.(type) {
	case int:
		return true
	case string:
		return true
	default:
		return false
	}
}

// Contract: upon return, each call to parse will have 
// head(tokens) be the first unparsed token in the sequence
func Parse(tokens *Tokens) (interface{}, error) {
	if len(tokens.tokens) == 0 { // this is dumb
		return nil, fmt.Errorf("cannot parse empty token sequence")
	}

	head := tokens.pop()

	for {
		switch head {
		case "(":
			elems := make([]interface{}, 0)

			for tokens.head() != ")" {
				n, err := Parse(tokens)
				if err != nil {
					return nil, fmt.Errorf("error parsing list: %w\n", err)
				}

				elems = append(elems, n)
			}
			// remove ')'
			tokens.pop()
			return elems, nil
		case ")":
			return nil, fmt.Errorf("unexpected ')'")
		default:
			atom, err := parseAtom(head)
			if err != nil {
				return nil, fmt.Errorf("error parsing atom: %w", err)
			}

			return atom, nil
		}
	}
}

func parseAtom(token string) (interface{}, error) {
	num, err := strconv.Atoi(token)

	if err == nil {
		return num, nil
	}

	return token, nil // we assume the token is a symbol
}

type Tokens struct {
	tokens []string
}

func (t *Tokens) pop() (string) {
	if len(t.tokens) == 0 {
		panic("cannot pop off empty list")
	}
	token := t.tokens[0]
	t.tokens = t.tokens[1:]

	return token
}

func (t *Tokens) head() (string) {
	if len(t.tokens) == 0 {
		panic("cannot pop off empty list")
	}
	return t.tokens[0]
}

func (t *Tokens) len() int {
	return len(t.tokens)
}

func Tokenize(code string) *Tokens {
	code = strings.ReplaceAll(code, "(", " ( ")
	code = strings.ReplaceAll(code, ")", " ) ")
	rawTokens := strings.Split(code, " ")

	tokens := make([]string, 0)

	for _, token := range rawTokens {
		if len(token) > 0 {
			tokens = append(tokens, token)
		}
	}

	return &Tokens{tokens}
}

