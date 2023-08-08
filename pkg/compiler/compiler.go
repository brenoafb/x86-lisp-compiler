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
)

type Compiler struct {
	W io.Writer
}

func (c *Compiler) compileExpr(expr interface{}) error {
	switch expr.(type) {
	case int:
		x := expr.(int)
		x <<= fixnumShift

		c.emit(fmt.Sprintf("movl $%d, %%eax", x))

		return nil

	case []interface{}:
		elems := expr.([]interface{})
		if len(elems) == 0 {
			c.emit(fmt.Sprintf("movl $%d, %%eax", emptyList))
			return nil
		}

		head := elems[0]

		if head == "add1" {
			x := elems[1]
			err := c.compileExpr(x)
			if err != nil {
				return fmt.Errorf("error compiling '%s' application: %w", "add1", err)
			}
			c.emit("addl $4, %eax")
			return nil
		}

		if head == "integer->char" {
			x := elems[1]
			err := c.compileExpr(x)
			if err != nil {
				return fmt.Errorf("error compiling '%s' application: %w", "integer->char", err)
			}
			c.emit(fmt.Sprintf("sall $%d, %%eax", charShift-fixnumShift))
			c.emit(fmt.Sprintf("orl $0x%x, %%eax", charTag))
			return nil
		}

		if head == "char->integer" {
			x := elems[1]
			err := c.compileExpr(x)
			if err != nil {
				return fmt.Errorf("error compiling '%s' application: %w", "char->integer", err)
			}
			c.emit(fmt.Sprintf("sarl $%d, %%eax", charShift-fixnumShift))
			return nil
		}

		if head == "null?" {
			return fmt.Errorf("null?: TODO")
		}

		if head == "zero?" {
			x := elems[1]
			err := c.compileExpr(x)
			if err != nil {
				return fmt.Errorf("error compiling '%s' application: %w", "zero?", err)
			}
			c.emit("cmpl $0, %eax")
			c.emit("movl $0, %eax")
			c.emit("sete %al")
			c.emit("sall $7, %eax")
			c.emit(fmt.Sprintf("orl $0x%x, %%eax", boolTag))

			return nil
		}

		return fmt.Errorf("unsupported operation %s", head)
	default:
		return fmt.Errorf("error compiling code: unsupported data type")
	}
}

func (c *Compiler) Compile(code string) error {
	tokens := parser.Tokenize(code)
	expr, err := parser.Parse(tokens)

	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	c.preamble()
	c.compileExpr(expr)
	c.emit("ret")

	return nil
}

func (c *Compiler) emit(code string) {
	fmt.Fprintln(c.W, code)
}

func (c *Compiler) preamble() {
	preamble := `    .text
    .globl  scheme_entry
    .p2align    2
scheme_entry:`
	c.emit(preamble)
}
