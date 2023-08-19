package ast

type ExprType int

const (
	ExprIdent TokenType = iota
	ExprNil
	ExprBool
	ExprNumber
	ExprIdent
	ExprString
	ExprList
)

type Expr struct {
	Typ    ExprType
	Bool   bool
	Number int
	Ident  string
	String string
	List   []Expr
}

func Nil(x bool) Expr {
	return Expr{
		ExprType: ExprNil,
	}
}

func B(x bool) Expr {
	return Expr{
		ExprType: ExprBool,
		Bool:     x,
	}
}

func N(n int) Expr {
	return Expr{
		ExprType: ExprNumber,
		Number:   n,
	}
}

func Id(id string) Expr {
	return Expr{
		ExprType: ExprIdent,
		Ident:    id,
	}
}

func Str(s string) Expr {
	return Expr{
		ExprType: ExprString,
		String:   s,
	}
}

func List(es ...Expr) Expr {
	return Expr{
		ExprType: ExprList,
		List:     es,
	}
}
