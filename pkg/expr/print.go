package expr

import (
	"fmt"
	"strings"
)

func (e *E) String() string {
	return e.prettyPrint(0) + "\n"
}

func (e *E) prettyPrint(level int) string {
	indent := strings.Repeat("  ", level) // 2 spaces per indentation level

	switch e.Typ {
	case ExprNil:
		return "()"
	case ExprIdent:
		return e.Ident
	case ExprBool:
		if e.Bool {
			return "true"
		}
		return "false"
	case ExprNumber:
		return fmt.Sprintf("%d", e.Number)
	case ExprString:
		return fmt.Sprintf("\"%s\"", e.Str)
	case ExprList:
		if len(e.List) == 0 {
			return "()"
		}
		listStr := "(" + e.List[0].prettyPrint(level)
		for i := 1; i < len(e.List); i++ {
			if e.List[i].Typ == ExprList {
				listStr += "\n" + indent + "  " + e.List[i].prettyPrint(level+1)
			} else {
				listStr += " " + e.List[i].prettyPrint(level+1)
			}
		}
		return listStr + ")"
	default:
		return "unknown_expr"
	}
}
