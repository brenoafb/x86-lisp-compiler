package compiler

import (
	"fmt"
	"io"

	"github.com/brenoafb/tinycompiler/pkg/expr"
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
	return &Compiler{
		W:   w,
		si:  -wordsize,
		env: make(map[string]location),
	}
}

func (c *Compiler) Compile(e expr.E) error {
	// we expect input to have format
	// (ident (<exported definitions>)
	//        (<constants>)
	//        (<internal procedures>)
	//   <body>)
	if e.Typ != expr.ExprList {
		return fmt.Errorf("input is no in expected format")
	}

	elems := e.List

	if len(elems) < 5 {
		return fmt.Errorf("top-level form must contain at least 5 elements")
	}

	if elems[0].Typ != expr.ExprIdent {
		// return fmt.Errorf(
		// 	"malformed top-level form: name is not ident",
		// )

		panic(fmt.Errorf(
			"malformed top-level form: name is not ident",
		))
	}

	topLevelName := elems[0].Ident

	if elems[1].Typ != expr.ExprList && elems[1].Typ != expr.ExprNil {
		return fmt.Errorf(
			"malformed top-level form: constants entry is not list",
		)
	}

	exports := elems[1].List

	if elems[2].Typ != expr.ExprList && elems[2].Typ != expr.ExprNil {
		return fmt.Errorf(
			"malformed top-level form: constants entry is not list",
		)
	}

	cvars := elems[2].List

	if elems[3].Typ != expr.ExprList && elems[3].Typ != expr.ExprNil {
		return fmt.Errorf(
			"malformed top-level form: procedures entry is not list",
		)
	}

	lvars := elems[3].List

	c.emit("\t.data")
	c.emit("\t.align\t8")

	for i, cvar := range cvars {
		if cvar.Typ != expr.ExprList && cvar.Typ != expr.ExprNil {
			return fmt.Errorf(
				"malformed top-level form: cvar at index %d is not list",
				i,
			)
		}
		pair := cvar.List
		if len(pair) != 2 {
			return fmt.Errorf("bad cvar in label form at index %d", i)
		}

		if pair[0].Typ != expr.ExprIdent {
			return fmt.Errorf(
				"malformed '_main' form: identifier at index %d is not identifier",
				i,
			)
		}

		name := pair[0].Ident
		cvarBody := pair[1]

		c.emit("%s:", name)

		err := c.compileExpr(cvarBody)
		if err != nil {
			return fmt.Errorf("error compiling body in _main form: %w", err)
		}
	}

	gatheredExports := make(map[string]expr.E)

	for i, export := range exports {
		if export.Typ != expr.ExprList && export.Typ != expr.ExprNil {
			return fmt.Errorf(
				"malformed top-level form: export at index %d is not list",
				i,
			)
		}
		tuple := export.List
		if len(tuple) != 2 {
			return fmt.Errorf("bad export in label form at index %d", i)
		}

		if tuple[0].Typ != expr.ExprIdent {
			return fmt.Errorf(
				"malformed 'label' form: identifier at index %d is not identifier",
				i,
			)
		}

		name := tuple[0].Ident
		body := tuple[1]

		gatheredExports[name] = body
	}

	c.emit("\t.text")
	c.emit("\t.p2align\t2")
	c.emit("\t.global %s", topLevelName)
	for name := range gatheredExports {
		c.emit("\t.global %s", name)
	}

	for name, body := range gatheredExports {
		c.emit("%s:", name)

		err := c.compileExpr(body)
		if err != nil {
			return fmt.Errorf("error compiling export body: %w", err)
		}
	}

	for i, lvar := range lvars {
		if lvar.Typ != expr.ExprList && lvar.Typ != expr.ExprNil {
			return fmt.Errorf(
				"malformed '_main' form: lvar at index %d is not list",
				i,
			)
		}
		pair := lvar.List
		if len(pair) != 2 {
			return fmt.Errorf("bad lvar in label form at index %d", i)
		}

		if pair[0].Typ != expr.ExprIdent {
			return fmt.Errorf(
				"malformed '_main' form: identifier at index %d is not identifier",
				i,
			)
		}

		name := pair[0].Ident
		lvarBody := pair[1]

		c.emit("%s:", name)

		err := c.compileExpr(lvarBody)
		if err != nil {
			return fmt.Errorf("error compiling body in _main form: %w", err)
		}
	}

	c.emit("%s:", topLevelName)
	c.emit("movl %%eax, %%esi")
	for _, body := range elems[4:] {
		err := c.compileExpr(body)
		if err != nil {
			return fmt.Errorf("error compiling body in _main form: %w", err)
		}
	}

	c.emit("ret")

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

			if _, ok := c.env[head.Ident]; ok {
				newExpr := []expr.E{
					expr.Id("funcall"),
				}

				for _, elem := range elems {
					newExpr = append(newExpr, elem)
				}

				return c.compileExpr(expr.L(newExpr...))
			}

			// assume the procedure is defined as a label
			newExpr := []expr.E{
				expr.Id("labelcall"),
			}

			for _, elem := range elems {
				newExpr = append(newExpr, elem)
			}

			return c.compileExpr(expr.L(newExpr...))
		case expr.ExprList:
			newExpr := []expr.E{
				expr.Id("funcall"),
			}

			for _, elem := range elems {
				newExpr = append(newExpr, elem)
			}

			return c.compileExpr(expr.L(newExpr...))
		}

		return fmt.Errorf("unsupported operation %s", head.String())
	default:
		return fmt.Errorf("error compiling code: %+v", e.String())
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
