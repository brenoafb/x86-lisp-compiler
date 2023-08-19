package parser

import (
	"fmt"
	"strconv"
	"unicode"
)

func pop(x []string) (string, []string) {
	if len(x) == 0 {
		panic("cannot pop off empty slice")
	}
	return x[0], x[1:]
}

// Contract: upon return, each call to parse will have
// head(tokens) be the first unparsed token in the sequence
func Parse(tokens *Tokens) (interface{}, error) {
	if len(tokens.tokens) == 0 { // this is dumb
		return nil, fmt.Errorf("cannot parse empty token sequence")
	}

	head := tokens.pop()

	for {
		switch head.Typ {
		case TokenIdent:
			return head.Ident, nil
		case TokenNumber:
			return head.Number, nil
		case TokenString:
			return head.String, nil
		case TokenRParen:
			return nil, fmt.Errorf("unexpected ')'")
		case TokenLParen:
			elems := make([]interface{}, 0)

			for tokens.head().Typ != TokenRParen {
				n, err := Parse(tokens)
				if err != nil {
					return nil, fmt.Errorf("error parsing list: %w\n", err)
				}

				elems = append(elems, n)
			}
			// remove ')'
			tokens.pop()
			return elems, nil
		}
	}
}

type TokenType int

const (
	TokenLParen TokenType = iota
	TokenRParen
	TokenIdent
	TokenNumber
	TokenString
)

type Token struct {
	Typ    TokenType
	Ident  string
	Number int
	String string
}

func lparen() Token {
	return Token{
		Typ: TokenLParen,
	}
}

func rparen() Token {
	return Token{
		Typ: TokenRParen,
	}
}

func ident(id string) Token {
	return Token{
		Typ:   TokenIdent,
		Ident: id,
	}
}

func number(n int) Token {
	return Token{
		Typ:    TokenNumber,
		Number: n,
	}
}

func str(s string) Token {
	return Token{
		Typ:    TokenString,
		String: s,
	}
}

type Tokens struct {
	tokens []Token
}

func (t *Tokens) pop() Token {
	if len(t.tokens) == 0 {
		panic("cannot pop off empty list")
	}
	token := t.tokens[0]
	t.tokens = t.tokens[1:]

	return token
}

func (t *Tokens) head() Token {
	if len(t.tokens) == 0 {
		panic("cannot pop off empty list")
	}
	return t.tokens[0]
}

func (t *Tokens) append(tok Token) {
	t.tokens = append(t.tokens, tok)
}

func (t *Tokens) len() int {
	return len(t.tokens)
}

func Tokenize(code string) (*Tokens, error) {
	runes := []rune(code)

	tokens := Tokens{
		tokens: make([]Token, 0),
	}

	i := 0

	for i < len(runes) {
		for unicode.IsSpace(runes[i]) {
			i++
			continue
		}

		if runes[i] == '(' {
			tokens.append(lparen())
			i++
			continue
		}

		if runes[i] == ')' {
			tokens.append(rparen())
			i++
			continue
		}

		var start int
		for start = i; unicode.IsDigit(runes[i]) && i < len(runes); i++ {
		}

		if start != i {
			// TODO handle malformed number
			// e.g. 123hello
			n, err := strconv.Atoi(string(runes[start:i]))
			if err != nil {
				return nil, fmt.Errorf(
					"error tokenizing number: %w",
					err,
				)
			}

			t := number(n)
			tokens.append(t)
			continue
		}

		for start = i; !unicode.IsSpace(runes[i]) &&
			runes[i] != '(' &&
			runes[i] != ')' &&
			i < len(runes); i++ {
		}

		if start != i {
			t := ident(string(runes[start:i]))
			tokens.append(t)
			continue
		}

	}

	return &tokens, nil
}
