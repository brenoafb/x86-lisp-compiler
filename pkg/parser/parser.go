package parser

import (
	"fmt"

	"github.com/brenoafb/tinycompiler/pkg/expr"
)

func pop(x []string) (string, []string) {
	if len(x) == 0 {
		panic("cannot pop off empty slice")
	}
	return x[0], x[1:]
}

func Parse(tokens *Tokens) ([]expr.E, error) {
	es := make([]expr.E, 0)
	for len(tokens.tokens) > 0 {
		e, err := parseExpr(tokens)
		if err != nil {
			return nil, fmt.Errorf("error parsing expressions: %w", err)
		}
		es = append(es, e)
	}

	return es, nil
}

// Contract: upon return, each call to parse will have
// head(tokens) be the first unparsed token in the sequence
func parseExpr(tokens *Tokens) (expr.E, error) {
	if len(tokens.tokens) == 0 {
		return expr.L(), fmt.Errorf("cannot parse empty token sequence")
	}

	head := tokens.pop()

	for {
		switch head.Typ {
		case TokenIdent:
			return expr.Id(head.Ident), nil
		case TokenNumber:
			return expr.N(head.Number), nil
		case TokenString:
			return expr.S(head.String), nil
		case TokenRParen:
			return expr.Nil(), fmt.Errorf("unexpected ')'")
		case TokenLParen:
			elems := make([]expr.E, 0)

			for tokens.head().Typ != TokenRParen {
				n, err := parseExpr(tokens)
				if err != nil {
					return expr.Nil(), fmt.Errorf("error parsing list: %w\n", err)
				}

				elems = append(elems, n)
			}
			// remove ')'
			tokens.pop()
			return expr.L(elems...), nil
		}
	}
}

