package preprocess

import (
	"fmt"

	"github.com/brenoafb/tinycompiler/pkg/expr"
)

func Preprocess(es []expr.E, name string) (expr.E, error) {
	for i, e := range es {
		e, err := annotateFreeVariables(e)
		if err != nil {
			return expr.Nil(), fmt.Errorf("preprocess: error annotating lambdas at index %d: %w", i, err)
		}
		es[i] = e
	}

	counter := 0
	lambdas := make(map[string]expr.E)

	var err error
	for i, e := range es {
		e, err = gatherLambdas(e, &counter, lambdas)

		if err != nil {
			return expr.Nil(), fmt.Errorf("preprocess: error gathering lambdas: %w", err)
		}
		es[i] = e
	}

	counter = 0
	strings := make(map[string]expr.E)

	for i, e := range es {
		e, err = gatherStrings(e, &counter, strings)

		if err != nil {
			return expr.Nil(), fmt.Errorf("preprocess: error gathering lambdas: %w", err)
		}

		es[i] = e
	}

	defuns := make(map[string]expr.E)
	es, err = gatherDefuns(es, defuns)

	constants := []expr.E{}

	for k, v := range strings {
		l := expr.L(
			expr.Id(k),
			v,
		)
		constants = append(constants, l)
	}

	labels := []expr.E{}

	for k, v := range lambdas {
		labels = append(labels, expr.L(
			expr.Id(k),
			v,
		))
	}

	exports := []expr.E{}

	for k, v := range defuns {
		exports = append(exports, expr.L(
			expr.Id(k),
			v,
		))
	}

	result := expr.L(
		expr.Id(name),
		expr.L(exports...),
		expr.L(constants...),
		expr.L(labels...),
	)

	for _, e := range es {
		result.List = append(result.List, e)
	}

	return result, nil
}

func annotateFreeVariables(
	e expr.E,
) (expr.E, error) {
	switch e.Typ {
	case expr.ExprList:
		elems := e.List
		if len(elems) == 0 {
			return e, nil
		}

		head := elems[0]

		if expr.IsIdent(head, "lambda") {
			if len(elems) != 3 {
				return expr.Nil(), fmt.Errorf("lambda form must contain 3 elements")
			}

			args := elems[1]
			if (args.Typ != expr.ExprList) && (args.Typ != expr.ExprNil) {
				return expr.Nil(), fmt.Errorf(
					"malformed lambda expression: args is not list %+v",
					args,
				)
			}

			body := elems[2]

			freeVars := make(map[string]struct{})
			argMap := make(map[string]struct{})
			for i, arg := range args.List {
				if arg.Typ != expr.ExprIdent {
					return expr.Nil(), fmt.Errorf(
						"malformed lambda expression: arg at index %d is not identifier",
						i,
					)
				}
				argMap[arg.Ident] = struct{}{}
			}

			err := gatherFreeVariables(body, argMap, freeVars)

			if err != nil {
				return expr.Nil(), fmt.Errorf("error annotating lambda expression: %w", err)
			}

			freeVarList := make([]expr.E, 0, len(freeVars))

			for k := range freeVars {
				freeVarList = append(freeVarList, expr.Id(k))
			}

			body, err = annotateFreeVariables(body)

			if err != nil {
				return expr.Nil(), fmt.Errorf(
					"error lifting free variables from lambda body: %w",
					err,
				)
			}

			newExpr := expr.L(
				expr.Id("lambda"),
				args,
				expr.L(freeVarList...),
				body,
			)

			return newExpr, nil
		}

		newExpr := make([]expr.E, 0, len(elems))

		for _, elem := range elems {
			elem, err := annotateFreeVariables(elem)
			if err != nil {
				return expr.Nil(), fmt.Errorf("error annotating free variables in sub expression: %w", err)
			}
			newExpr = append(newExpr, elem)
		}

		return expr.L(newExpr...), nil
	default:
		return e, nil
	}
}

func gatherFreeVariables(
	e expr.E,
	args map[string]struct{},
	freeVars map[string]struct{},
) error {
	switch e.Typ {
	case expr.ExprIdent:
		v := e.Ident
		if _, ok := builtins[v]; ok {
			return nil
		}
		if _, ok := args[v]; ok {
			return nil
		}

		freeVars[v] = struct{}{}

		return nil

	case expr.ExprList:
		elems := e.List
		if len(elems) == 0 {
			return nil
		}

		for _, elem := range elems {
			err := gatherFreeVariables(elem, args, freeVars)
			if err != nil {
				return fmt.Errorf("error gathering free vars from subexpression: %w", err)
			}
		}
	default:
		return nil
	}

	return nil
}

func gatherLambdas(
	e expr.E,
	counter *int,
	lambdas map[string]expr.E,
) (expr.E, error) {
	switch e.Typ {
	case expr.ExprList:
		elems := e.List
		if expr.IsIdent(elems[0], "lambda") {
			args := elems[1]
			freeVars := elems[2]
			body := elems[3]

			if freeVars.Typ != expr.ExprList && freeVars.Typ != expr.ExprNil {
				return expr.Nil(), fmt.Errorf("malformed lambda form")
			}

			var err error
			body, err = gatherLambdas(body, counter, lambdas)
			if err != nil {
				return expr.Nil(), fmt.Errorf("error gathering lambdas recursively: %w", err)
			}

			k := *counter
			*counter = *counter + 1
			label := fmt.Sprintf("f%d", k)

			newExpr := []expr.E{
				expr.Id("closure"),
				expr.Id(label),
			}

			for _, freeVar := range freeVars.List {
				newExpr = append(newExpr, freeVar)
			}

			code := expr.L(
				expr.Id("code"),
				args,
				freeVars,
				body,
			)

			lambdas[label] = code
			return expr.L(newExpr...), nil
		}

		newExpr := make([]expr.E, 0, len(elems))

		for _, elem := range elems {
			elem, err := gatherLambdas(elem, counter, lambdas)
			if err != nil {
				return expr.Nil(), fmt.Errorf("error gathering lambdas in sub expression: %w", err)
			}
			newExpr = append(newExpr, elem)
		}

		return expr.L(newExpr...), nil

	default:
		return e, nil
	}
}

func gatherStrings(
	e expr.E,
	counter *int,
	strings map[string]expr.E,
) (expr.E, error) {
	switch e.Typ {
	case expr.ExprString:
		k := *counter
		*counter = *counter + 1
		label := fmt.Sprintf("s%d", k)

		newExpr := []expr.E{
			expr.Id("string-ref"),
			expr.Id(label),
		}

		strings[label] = expr.L(
			expr.Id("string-init"),
			e,
		)

		return expr.L(newExpr...), nil
	case expr.ExprList:
		elems := e.List
		newExpr := make([]expr.E, 0, len(elems))

		for _, elem := range elems {
			elem, err := gatherStrings(elem, counter, strings)
			if err != nil {
				return expr.Nil(), fmt.Errorf("gathering strings sub expression: %w", err)
			}
			newExpr = append(newExpr, elem)
		}

		return expr.L(newExpr...), nil

	default:
		return e, nil
	}
}

func gatherDefuns(
	es []expr.E,
	defuns map[string]expr.E,
) ([]expr.E, error) {
	for i, e := range es {
		switch e.Typ {
		case expr.ExprList:
			elems := e.List
			if expr.IsIdent(elems[0], "defun") {
				name := elems[1]
				args := elems[2]
				body := elems[3]

				if name.Typ != expr.ExprIdent {
					return nil, fmt.Errorf("malformed defun form")
				}

				if args.Typ != expr.ExprList && args.Typ != expr.ExprNil {
					return nil, fmt.Errorf("malformed defun form")
				}

				code := expr.L(
					expr.Id("code"),
					args,
					expr.L(),
					body,
				)

				defuns[name.Ident] = code

				es[i] = expr.Nil()
				continue
			}

			es[i] = e
		default:
			es[i] = e
		}
	}
	return es, nil
}
