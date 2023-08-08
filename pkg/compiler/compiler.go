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
	wordsize    = 8
)

type Compiler struct {
	W  io.Writer
	si int
}

func NewCompiler(w io.Writer) *Compiler {
	return &Compiler{W: w, si: -wordsize}
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
			c.emit(fmt.Sprintf("movl $0x%x, %%eax", emptyList))
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
			x := elems[1]
			err := c.compileExpr(x)
			if err != nil {
				return fmt.Errorf("error compiling '%s' application: %w", "zero?", err)
			}
			c.emit(fmt.Sprintf("cmpl $0x%x, %%eax", emptyList))
			c.emit("movl $0, %eax")
			c.emit("sete %al")
			c.emit("sall $7, %eax")
			c.emit(fmt.Sprintf("orl $0x%x, %%eax", boolTag))

			return nil
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

		if head == "+" {
			x := elems[1]
			y := elems[2]
			err := c.compileExpr(y)
			if err != nil {
				return fmt.Errorf("error compiling '%s' application: %w", "+", err)
			}
			c.push()
			err = c.compileExpr(x)
			c.si += wordsize
			if err != nil {
				return fmt.Errorf("error compiling '%s' application: %w", "+", err)
			}
			c.emit(fmt.Sprintf("addl %d(%%rsp), %%eax", c.si))

			return nil
		}

		if head == "*" {
			x := elems[1]
			y := elems[2]
			err := c.compileExpr(y)
			if err != nil {
				return fmt.Errorf("error compiling '%s' application: %w", "+", err)
			}
			c.push()
			err = c.compileExpr(x)
			c.si += wordsize
			if err != nil {
				return fmt.Errorf("error compiling '%s' application: %w", "+", err)
			}
			c.emit(fmt.Sprintf("imull %d(%%rsp), %%eax", c.si))

			return nil
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
	c.emit(fmt.Sprintf("movl %%eax, %d(%%rsp)", c.si))
	c.si -= wordsize
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
