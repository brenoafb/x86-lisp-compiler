package compiler

import (
	"fmt"
	"io"

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
	tokens := parser.Tokenize(code)
	expr, err := parser.Parse(tokens)

	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	c.preamble()
	c.compileExpr(expr)

	return nil
}

func (c *Compiler) preprocess(expr interface{}) (interface{}, error) {
	return []interface{}{
		"labels",
		[]interface{}{},
		expr,
	}, nil
}

func (c *Compiler) annotateFreeVariables(
	expr interface{},
	args map[string]struct{},
	freeVars map[string]struct{},
) ([]string, error) {
	switch expr.(type) {
	case string:
		// TODO
		return nil, nil
	case []interface{}:
		elems := expr.([]interface{})
		if len(elems) == 0 {
			return nil, nil
		}

		head := elems[0]

		if head == "lambda" {
			if len(elems) != 3 {
				return nil, fmt.Errorf("lambda form must contain 3 elements")
			}

			// args := elems[1].([]interface{})
		}
	}

	return nil, fmt.Errorf("not implemented")
}

func (c *Compiler) gatherFreeVariables(
	expr interface{},
	args map[string]struct{},
	freeVars map[string]struct{},
) error {
	switch expr.(type) {
	case string:
		v := expr.(string)
		if _, ok := args[v]; ok {
			return nil
		}

		freeVars[v] = struct{}{}

		return nil

	case []interface{}:
		elems := expr.([]interface{})
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

func (c *Compiler) compileExpr(expr interface{}) error {
	switch expr.(type) {
	case string:
		v := expr.(string)
		loc, ok := c.env[v]
		if !ok {
			return fmt.Errorf("unbound variable '%s'", v)
		}
		switch loc.location {
		case stack:
			c.emit(fmt.Sprintf("movl %d(%%esp), %%eax", loc.offset))
		case closure:
			c.emit(fmt.Sprintf("movl %d(%%edi), %%eax", loc.offset))
		case heap:
			c.emit(fmt.Sprintf("movl %d(%%esi), %%eax", loc.offset))
		}
		return nil
	case int:
		x := expr.(int)
		x <<= fixnumShift

		c.emit(fmt.Sprintf("movl $%d, %%eax", x))

		return nil

	case []interface{}:
		elems := expr.([]interface{})
		if len(elems) == 0 {
			c.emit(fmt.Sprintf("movl $0x%x, %%eax", emptyList))
			return nil
		}

		head := elems[0]

		switch head.(type) {
			case string:
			if proc, ok := builtins[head.(string)]; ok {
				return proc(c, elems)
			}
		}

		return fmt.Errorf("unsupported operation %s", head)
	default:
		return fmt.Errorf("error compiling code: unsupported data type")
	}
}

// push %eax onto the stack
func (c *Compiler) push() {
	// si points to the top of the stack
	// i.e. in the free space above the stack frame
	c.emit(fmt.Sprintf("movl %%eax, %d(%%esp)", c.si))
	c.si -= wordsize
}

func (c *Compiler) clearEnv() {
	c.env = make(map[string]location)
}

func (c *Compiler) emit(code string) {
	fmt.Fprintln(c.W, code)
}

func (c *Compiler) preamble() {
	preamble := `    .text
    .globl  scheme_entry
    .p2align    2
scheme_entry:
movl %eax, %esi`
	c.emit(preamble)
}

func (c *Compiler) genLabel() string {
	n := c.labelCounter
	c.labelCounter++
	return fmt.Sprintf("L%d", n)
}
