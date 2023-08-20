package expr

type ExprType int

const (
	ExprNil ExprType = iota
	ExprIdent
	ExprBool
	ExprNumber
	ExprString
	ExprList
)

type E struct {
	Typ    ExprType
	Bool   bool
	Number int
	Ident  string
	Str    string
	List   []E
}

func Nil() E {
	return E{
		Typ: ExprNil,
	}
}

func B(x bool) E {
	return E{
		Typ:  ExprBool,
		Bool: x,
	}
}

func N(n int) E {
	return E{
		Typ:    ExprNumber,
		Number: n,
	}
}

func Id(id string) E {
	return E{
		Typ:   ExprIdent,
		Ident: id,
	}
}

func S(s string) E {
	return E{
		Typ: ExprString,
		Str: s,
	}
}

func L(es ...E) E {
	if len(es) == 0 {
		return Nil()
	}
	return E{
		Typ:  ExprList,
		List: es,
	}
}

func IsIdent(e E, s string) bool {
	return e.Typ == ExprIdent && e.Ident == s
}
