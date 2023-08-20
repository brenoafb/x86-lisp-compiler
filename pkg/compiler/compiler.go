package compiler

import (
	"fmt"
	"io"

	"github.com/brenoafb/tinycompiler/pkg/expr"
	"github.com/brenoafb/tinycompiler/pkg/parser"
	pp "github.com/brenoafb/tinycompiler/pkg/preprocess"
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
		W: w, 
		si: -wordsize, 
		env: make(map[string]location),
	}
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

	e, err = pp.Preprocess(e)

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
