package parser

import (
	"fmt"
	"strconv"
	"unicode"
)

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
		for i < len(runes) && unicode.IsSpace(runes[i]) {
			i++
			continue
		}

		if i >= len(runes) {
			break
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

		if runes[i] == '"' {
			i++
			for start = i; i < len(runes) && runes[i] != '"'; i++ {
			}
			if i == len(runes) {
				return nil, fmt.Errorf("Input ended before string literal terminated")
			}

			t := str(string(runes[start:i]))
			tokens.append(t)
			i++
			continue
		}

		for start = i; i < len(runes) && unicode.IsDigit(runes[i]); i++ {
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

		for start = i; i < len(runes) &&
			!unicode.IsSpace(runes[i]) &&
			runes[i] != '(' &&
			runes[i] != ')'; i++ {
		}

		if start != i {
			t := ident(string(runes[start:i]))
			tokens.append(t)
			continue
		}

	}

	return &tokens, nil
}
