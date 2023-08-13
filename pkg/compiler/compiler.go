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

		if head == "let" {
			// (let <bindings...> <body>)
			if len(elems) < 3 {
				panic("invalid let form")
			}
			si := c.si
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
				c.env[v] = location{
					location: stack,
					offset:   idx,
				}
			}

			err := c.compileExpr(body)
			if err != nil {
				return fmt.Errorf("error compiling let binding body: %w", err)
			}

			c.si = si

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
			c.emit(fmt.Sprintf("addl %%ebx, %%esi"))
			return nil
		}

		if head == "vector-ref" {
			if len(elems) != 3 {
				return fmt.Errorf("vector-ref requires 2 parameters")
			}

			vector := elems[1]
			idx := elems[2]

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
			idx := elems[2]
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

		if head == "labels" {
			if len(elems) != 3 {
				return fmt.Errorf("labels form must contain 3 elements")
			}

			lvars := elems[1].([]interface{})
			body := elems[2]

			err := c.compileExpr(body)
			if err != nil {
				return fmt.Errorf("error compiling body in labels form: %w", err)
			}

			c.emit("ret")

			for i, lvar := range lvars {
				pair := lvar.([]interface{})
				if len(pair) != 2 {
					return fmt.Errorf("bad lvar in label form at index %d", i)
				}
				name := pair[0].(string)
				lvarBody := pair[1]

				c.emit(fmt.Sprintf("%s:", name))

				err := c.compileExpr(lvarBody)
				if err != nil {
					return fmt.Errorf("error compiling body in labels form: %w")
				}
			}
		}

		if head == "code" {
			if len(elems) != 4 {
				return fmt.Errorf("'code' form must contain 3 parameters")
			}

			arglist := elems[1].([]interface{})
			freevars := elems[2].([]interface{})
			body := elems[3]

			// assign stack location for each argument
			for i, arg := range arglist {
				v := arg.(string)
				c.env[v] = location{
					location: stack,
					offset:   -wordsize * (i + 1),
				}
			}

			// adjust the stack so that we point on top
			// of the arguments
			c.si = -wordsize * (len(arglist) + 1)

			// assign closure location for each free variable
			for i, arg := range freevars {
				fv := arg.(string)
				c.env[fv] = location{
					location: closure,
					offset:   -wordsize * (i + 1),
				}
			}

			err := c.compileExpr(body)
			if err != nil {
				return fmt.Errorf("error compiling body in 'code' form: %w")
			}

			c.emit("ret")
			c.clearEnv()
			c.si = -wordsize
			return nil
		}

		if head == "labelcall" {
			if len(elems) < 2 {
				return fmt.Errorf("labelcall form must contain at least 1 parameter")
			}

			l := elems[1].(string)
			spSlot := c.si + wordsize
			siBefore := c.si
			// skip one slot for the return address
			c.si -= wordsize
			for i, arg := range elems[2:] {
				err := c.compileExpr(arg)
				if err != nil {
					return fmt.Errorf("error compiling argument at index %d in labelcall: %w", i, err)
				}
				c.push()
			}
			// handle call and return
			// call subtracts wordsize from esp, so we need to adjust it first
			// to make sure we don't overwrite local variables
			c.emit(fmt.Sprintf("addl $%d, %%esp", spSlot))
			c.emit(fmt.Sprintf("call %s", l))
			// restore esp
			c.emit(fmt.Sprintf("addl $%d, %%esp", -spSlot))
			c.si = siBefore

			return nil
		}

		if head == "funcall" {
			if len(elems) < 2 {
				return fmt.Errorf("funcall form must contain at least 1 parameter")
			}

			f := elems[1]

			spSlot := c.si
			siBefore := c.si
			// skip two slots for the return address and closure pointer
			c.si -= 2 * wordsize
			for i, arg := range elems[2:] {
				err := c.compileExpr(arg)
				if err != nil {
					return fmt.Errorf("error compiling argument at index %d in funcall: %w", i, err)
				}
				c.push()
			}

			// save closure pointer
			c.emit(fmt.Sprintf("movl %%edi, %d(%%esp)", siBefore))

			err := c.compileExpr(f)
			if err != nil {
				return fmt.Errorf("error compiling function in funcall: %w", err)
			}

			// move new closure into closure pointer
			c.emit("movl %eax, %edi")

			l := "f0" // TODO read label from closure pointer

			// handle call and return
			c.emit(fmt.Sprintf("addl $%d, %%esp", spSlot))
			c.emit(fmt.Sprintf("call %s", l))
			c.emit(fmt.Sprintf("addl $%d, %%esp", -spSlot))
			c.si = siBefore

			return nil
		}

		if head == "closure" {
			if len(elems) < 2 {
				return fmt.Errorf("closure form must contain at least 1 parameter")
			}

			// TODO push label to first location at closure pointer
			// l := elems[1].(string)
			c.emit(fmt.Sprintf("movl $0, 0(%%esi)"))
			for i, freevar := range elems[2:] {
				// TODO copy free variable value directly instead
				// of moving to eax then to heap
				err := c.compileExpr(freevar.(string))
				if err != nil {
					return fmt.Errorf(
						"error compiling free variable in closure form at index %d: %w",
						i,
						err,
					)
				}

				c.emit(fmt.Sprintf("movl %%eax, %d(%%esi)", wordsize*(i+1)))
			}

			length := len(elems) - 2

			c.emit(fmt.Sprintf("movl $%d, %%ebx", length))
			// eax = esi | 6
			c.emit(fmt.Sprintf("movl %%esi, %%eax"))
			c.emit(fmt.Sprintf("orl $6, %%eax")) // 6 = closure tag
			// align size to next object boundary
			c.emit(fmt.Sprintf("addl $11, %%ebx"))
			c.emit(fmt.Sprintf("andl $-8, %%ebx"))
			// advance alloc ptr
			c.emit(fmt.Sprintf("addl %%ebx, %%esi"))
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
