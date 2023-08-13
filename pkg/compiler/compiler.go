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
	env          map[string]int
	labelCounter int
}

func NewCompiler(w io.Writer) *Compiler {
	env := make(map[string]int)
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
	c.emit("ret")

	return nil
}

func (c *Compiler) compileExpr(expr interface{}) error {
	switch expr.(type) {
	case string:
		v := expr.(string)
		idx, ok := c.env[v]
		if !ok {
			return fmt.Errorf("unbound variable '%s'", v)
		}
		c.emit(fmt.Sprintf("movl %d(%%esp), %%eax", idx))
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
			c.emit(fmt.Sprintf("addl %d(%%esp), %%eax", c.si))

			return nil
		}

		if head == "-" {
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
			c.emit(fmt.Sprintf("subl %d(%%esp), %%eax", c.si))

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
			if err != nil {
				return fmt.Errorf("error compiling '%s' application: %w", "+", err)
			}
			c.si += wordsize
			c.emit(fmt.Sprintf("imull %d(%%esp), %%eax", c.si))

			return nil
		}

		if head == "let" {
			// (let <bindings...> <body>)
			if len(elems) < 3 {
				panic("invalid let form")
			}
			bindings := elems[1 : len(elems)-1]
			body := elems[len(elems)-1]

			// each binding has form
			// (<variable> <body>)
			for _, binding := range bindings {
				xs := binding.([]interface{})
				v := xs[0].(string)
				b := xs[1]
				err := c.compileExpr(b)
				if err != nil {
					return fmt.Errorf("error compiling let binding: %w", err)
				}
				idx := c.si
				c.push()
				c.env[v] = idx
			}

			err := c.compileExpr(body)
			if err != nil {
				return fmt.Errorf("error compiling let binding body: %w", err)
			}

			return nil
		}

		if head == "if" {
			if len(elems) != 4 {
				return fmt.Errorf("malformed 'if' expression")
			}
			test := elems[1]
			conseq := elems[2]
			alt := elems[3]

			l0 := c.genLabel()
			l1 := c.genLabel()

			err := c.compileExpr(test)
			if err != nil {
				return fmt.Errorf("error compiling test in if expression: %w", err)
			}
			c.emit(fmt.Sprintf("cmpl $0x%x, %%eax", immFalse))
			c.emit(fmt.Sprintf("je %s", l0))

			err = c.compileExpr(conseq)
			if err != nil {
				return fmt.Errorf("error compiling conseq in if expression: %w", err)
			}

			c.emit(fmt.Sprintf("jmp %s", l1))
			c.emit(fmt.Sprintf("%s:", l0))

			err = c.compileExpr(alt)
			if err != nil {
				return fmt.Errorf("error compiling alt in if expression: %w", err)
			}

			c.emit(fmt.Sprintf("%s:", l1))
		}

		if head == "cons" {
			if len(elems) != 3 {
				return fmt.Errorf("malformed cons expression")
			}
			x := elems[1]
			y := elems[2]

			err := c.compileExpr(x)
			if err != nil {
				return fmt.Errorf("error compiling cons expression: %w", err)
			}
			c.emit(fmt.Sprintf("movl %%eax, %d(%%esi)", 0*wordsize))

			err = c.compileExpr(y)
			if err != nil {
				return fmt.Errorf("error compiling cons expression: %w", err)
			}
			c.emit(fmt.Sprintf("movl %%eax, %d(%%esi)", 1*wordsize))

			c.emit("movl %esi, %eax")
			c.emit("orl $1, %eax")

			c.emit(fmt.Sprintf("addl $%d, %%esi", 2*wordsize))

			return nil
		}

		if head == "car" {
			if len(elems) != 2 {
				return fmt.Errorf("malformed car expression")
			}
			err := c.compileExpr(elems[1])
			if err != nil {
				return fmt.Errorf("error compiling car expression: %w", err)
			}

			c.emit("movl -1(%eax), %eax")
			return nil
		}

		if head == "cdr" {
			if len(elems) != 2 {
				return fmt.Errorf("malformed cdr expression")
			}
			err := c.compileExpr(elems[1])
			if err != nil {
				return fmt.Errorf("error compiling car expression: %w", err)
			}

			c.emit(fmt.Sprintf("movl %d(%%eax), %%eax", wordsize-1))
			return nil
		}

		if head == "make-vector" {
			if len(elems) != 2 {
				return fmt.Errorf("malformed make-vector expression")
			}

			err := c.compileExpr(elems[1])

			if err != nil {
				return fmt.Errorf("error compiling make-vector call: %w", err)
			}

			// set length
			c.emit(fmt.Sprintf("movl %%eax, 0(%%esi)"))
			// save length
			c.emit(fmt.Sprintf("movl %%eax, %%ebx"))
			// eax = esi | 2
			c.emit(fmt.Sprintf("movl %%esi, %%eax"))
			c.emit(fmt.Sprintf("orl $2, %%eax")) // 2 = vector tag
			// align size to next object boundary
			c.emit(fmt.Sprintf("addl $11, %%ebx"))
			c.emit(fmt.Sprintf("andl $-8, %%ebx"))
			// advance alloc ptr
			c.emit(fmt.Sprintf("andl %%ebx, %%esi"))
			return nil
		}

		if head == "vector-ref" {
			if len(elems) != 3 {
				return fmt.Errorf("vector-ref requires 2 parameters")
			}

			vector := elems[1]
			idx:= elems[2]

			err := c.compileExpr(vector)
			if err != nil {
				return fmt.Errorf("error compiling vector expr in vector-ref call: %w", err)
			}

			// save vector ptr
			vectorIdx := c.si
			c.push()

			err = c.compileExpr(idx)
			if err != nil {
				return fmt.Errorf("error compiling index expr in vector-ref call: %w", err)
			}

			c.emit("addl $1, %eax")
			c.emit(fmt.Sprintf("movl %d(%%esp), %%ebx", vectorIdx))
			c.emit("addl %ebx, %eax")
			c.emit("movl 0(%eax), %eax")

			return nil
		}

		if head == "vector-set!" {
			if len(elems) != 4 {
				return fmt.Errorf("vector-set requires 3 parameters")
			}

			vector := elems[1]
			idx:= elems[2]
			obj := elems[3]


			err := c.compileExpr(vector)
			if err != nil {
				return fmt.Errorf("error compiling vector expr in vector-set! call: %w", err)
			}

			// save vector ptr
			vectorIdx := c.si
			c.push()

			err = c.compileExpr(idx)
			if err != nil {
				return fmt.Errorf("error compiling index expr in vector-set! call: %w", err)
			}

			// save idx
			idxIdx := c.si
			c.push()

			err = c.compileExpr(obj)
			if err != nil {
				return fmt.Errorf("error compiling object expr in vector-set! call: %w", err)
			}

			// compute destination pointer
			c.emit(fmt.Sprintf("movl %d(%%esp), %%ebx", idxIdx))
			c.emit("addl $1, %ebx")
			c.emit(fmt.Sprintf("addl %d(%%esp), %%ebx", vectorIdx))
			// move object into slot
			c.emit("movl %eax, 0(%ebx)")
			// set eax to vector pointer
			c.emit(fmt.Sprintf("movl %d(%%esp), %%eax", vectorIdx))

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
	c.emit(fmt.Sprintf("movl %%eax, %d(%%esp)", c.si))
	c.si -= wordsize
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
	return fmt.Sprintf(".L%d", n)
}
