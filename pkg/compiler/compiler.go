package compiler

import (
	"fmt"
	"io"

	"github.com/brenoafb/tinycompiler/pkg/expr"
	"github.com/brenoafb/tinycompiler/pkg/parser"
)

const (
	fixnumShift = 2
	fixnumTag   = 0
	charShift   = 8
	charTag     = 0x0f
	emptyList   = 0x2f
	boolTag     = 0x1f
	immFalse    = 0x1f
	immTrue     = 0x9f
	wordsize    = 4
)

type Compiler struct {
	W            io.Writer
	si           int
	env          map[string]location
	labelCounter int
}

type memlocation int

const (
	stack memlocation = iota
	closure
	heap
)

type location struct {
	location memlocation
	offset   int
}

func NewCompiler(w io.Writer) *Compiler {
	env := make(map[string]location)
	return &Compiler{W: w, si: -wordsize, env: env}
}

func (c *Compiler) Compile(code string) error {
	tokens, err := parser.Tokenize(code)

	if err != nil {
		return fmt.Errorf("tokenizer error: %w", err)
	}

	e, err := parser.Parse(tokens)

	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	e, err = c.preprocess(e)

	if err != nil {
		return fmt.Errorf("preprocessor error: %w", err)
	}

	fmt.Printf("%s\n", e.String())

	err = c.compileExpr(e)

	if err != nil {
		return fmt.Errorf("error compiling expression: %w", err)
	}

	return nil
}

func (c *Compiler) gatherLambdas(
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
			body, err = c.gatherLambdas(body, counter, lambdas)
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
			elem, err := c.gatherLambdas(elem, counter, lambdas)
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

func (c *Compiler) gatherStrings(
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
			elem, err := c.gatherStrings(elem, counter, strings)
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

func (c *Compiler) preprocess(e expr.E) (expr.E, error) {
	e, err := c.annotateFreeVariables(e)

	if err != nil {
		return expr.Nil(), fmt.Errorf("preprocess: error annotating lambdas: %w", err)
	}

	counter := 0
	lambdas := make(map[string]expr.E)

	e, err = c.gatherLambdas(e, &counter, lambdas)

	if err != nil {
		return expr.Nil(), fmt.Errorf("preprocess: error gathering lambdas: %w", err)
	}

	counter = 0
	strings := make(map[string]expr.E)

	e, err = c.gatherStrings(e, &counter, strings)

	if err != nil {
		return expr.Nil(), fmt.Errorf("preprocess: error gathering lambdas: %w", err)
	}

	constants := []expr.E{}

	for k, v := range strings {
		constants = append(constants, expr.L(
			expr.Id(k),
			v,
		))
	}

	labels := []expr.E{}

	for k, v := range lambdas {
		labels = append(labels, expr.L(
			expr.Id(k),
			v,
		))
	}

	result := expr.L(
		expr.Id("_main"),
		expr.L(constants...),
		expr.L(labels...),
		e,
	)

	return result, nil
}

func (c *Compiler) annotateFreeVariables(
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

			err := c.gatherFreeVariables(body, argMap, freeVars)

			if err != nil {
				return expr.Nil(), fmt.Errorf("error annotating lambda expression: %w", err)
			}

			freeVarList := make([]expr.E, 0, len(freeVars))

			for k := range freeVars {
				freeVarList = append(freeVarList, expr.Id(k))
			}

			body, err = c.annotateFreeVariables(body)

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
			elem, err := c.annotateFreeVariables(elem)
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

func (c *Compiler) gatherFreeVariables(
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
			err := c.gatherFreeVariables(elem, args, freeVars)
			if err != nil {
				return fmt.Errorf("error gathering free vars from subexpression: %w", err)
			}
		}
	default:
		return nil
	}

	return nil
}

func (c *Compiler) compileExpr(e expr.E) error {
	switch e.Typ {
	case expr.ExprIdent:
		v := e.Ident
		loc, ok := c.env[v]
		if !ok {
			return fmt.Errorf("unbound variable '%s'", v)
		}
		switch loc.location {
		case stack:
			c.emit("movl %d(%%esp), %%eax", loc.offset)
		case closure:
			c.emit("movl %d(%%edi), %%eax", loc.offset)
		case heap:
			c.emit("movl %d(%%esi), %%eax", loc.offset)
		}
		return nil
	case expr.ExprNumber:
		x := e.Number
		x <<= fixnumShift

		c.emit("movl $%d, %%eax", x)

		return nil
	case expr.ExprNil:
		c.emit("movl $0x%x, %%eax", emptyList)
		return nil

	case expr.ExprList:
		elems := e.List
		if len(elems) == 0 {
			c.emit("movl $0x%x, %%eax", emptyList)
			return nil
		}

		head := elems[0]

		switch head.Typ {
		case expr.ExprIdent:
			if proc, ok := builtins[head.Ident]; ok {
				return proc(c, elems)
			}
		case expr.ExprList:
			newExpr := []expr.E{
				expr.Id("funcall"),
			}

			for _, elem := range elems {
				newExpr = append(newExpr, elem)
			}

			return c.compileExpr(expr.L(newExpr...))
		}

		return fmt.Errorf("unsupported operation %v", head)
	default:
		// return fmt.Errorf("error compiling code: %+v", e.String())
		panic(fmt.Errorf("error compiling code: %+v", e.String()))
	}
}

// push %eax onto the stack
func (c *Compiler) push() {
	// si points to the top of the stack
	// i.e. in the free space above the stack frame
	c.emit("movl %%eax, %d(%%esp)", c.si)
	c.si -= wordsize
}

func (c *Compiler) clearEnv() {
	c.env = make(map[string]location)
}

func (c *Compiler) emit(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Fprintln(c.W, s)
}

func (c *Compiler) genLabel() string {
	n := c.labelCounter
	c.labelCounter++
	return fmt.Sprintf("L%d", n)
}
