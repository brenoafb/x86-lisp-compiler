package parser

import (
	"fmt"

	"github.com/brenoafb/tinycompiler/pkg/ast"
)

func pop(x []string) (string, []string) {
	if len(x) == 0 {
		panic("cannot pop off empty slice")
	}
	return x[0], x[1:]
}

// Contract: upon return, each call to parse will have
// head(tokens) be the first unparsed token in the sequence
func Parse(tokens *Tokens) (ast.Expr, error) {
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
			elems := make([]ast.Expr, 0)

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
