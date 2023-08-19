package compiler

import (
	"fmt"

	"github.com/brenoafb/tinycompiler/pkg/expr"
)

type builtin func(c *Compiler, elems []expr.E) error

var builtins map[string]builtin

func init() {
	builtins = map[string]builtin{
		// special forms
		"progn": func(c *Compiler, elems []expr.E) error {
			for i, expr := range elems[1:] {
				err := c.compileExpr(expr)
				if err != nil {
					return fmt.Errorf(
						"error compiling progn expression at index %d: %w",
						i,
						err,
					)
				}
			}
			return nil
		},
		"define": func(c *Compiler, elems []expr.E) error {
			return fmt.Errorf("'define' is not supported")
		},
		"let": func(c *Compiler, elems []expr.E) error {
			// (let <bindings...> <body>)
			if len(elems) < 3 {
				panic("invalid let form")
			}
			si := c.si
			bindings := elems[1 : len(elems)-1]
			body := elems[len(elems)-1]

			// each binding has form
			// (<variable> <body>)
			for i, binding := range bindings {
				if binding.Typ != expr.ExprList && binding.Typ != expr.ExprNil {
					return fmt.Errorf(
						"error compiling let binding: element at index %d is not list",
						i,
					)
				}
				xs := binding.List
				if xs[0].Typ != expr.ExprIdent {
					return fmt.Errorf(
						"error compiling let binding: variable at index %d is not identifier",
						i,
					)
				}
				v := xs[0].Ident
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
		},

		"if": func(c *Compiler, elems []expr.E) error {
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
			c.emit("cmpl $0x%x, %%eax", immFalse)
			c.emit("je %s", l0)

			err = c.compileExpr(conseq)
			if err != nil {
				return fmt.Errorf("error compiling conseq in if expression: %w", err)
			}

			c.emit("jmp %s", l1)
			c.emit("%s:", l0)

			err = c.compileExpr(alt)
			if err != nil {
				return fmt.Errorf("error compiling alt in if expression: %w", err)
			}

			c.emit("%s:", l1)

			return nil
		},
		"labels": func(c *Compiler, elems []expr.E) error {
			if len(elems) != 3 {
				return fmt.Errorf("labels form must contain 3 elements")
			}

			if elems[1].Typ != expr.ExprList && elems[1].Typ != expr.ExprNil {
				return fmt.Errorf(
					"malformed 'labels' form: lvars is not list",
				)
			}
			lvars := elems[1].List
			body := elems[2]

			err := c.compileExpr(body)
			if err != nil {
				return fmt.Errorf("error compiling body in labels form: %w", err)
			}

			c.emit("ret")

			for i, lvar := range lvars {
				if lvar.Typ != expr.ExprList && lvar.Typ != expr.ExprNil {
					return fmt.Errorf(
						"malformed 'labels' form: lvar at index %d is not list",
						i,
					)
				}
				pair := lvar.List
				if len(pair) != 2 {
					return fmt.Errorf("bad lvar in label form at index %d", i)
				}

				if pair[0].Typ != expr.ExprIdent {
					return fmt.Errorf(
						"malformed 'labels' form: identifier at index %d is not identifier",
						i,
					)
				}

				name := pair[0].Ident
				lvarBody := pair[1]

				c.emit("%s:", name)

				err := c.compileExpr(lvarBody)
				if err != nil {
					return fmt.Errorf("error compiling body in labels form: %w", err)
				}
			}

			return nil
		},

		"code": func(c *Compiler, elems []expr.E) error {
			if len(elems) != 4 {
				return fmt.Errorf("'code' form must contain 3 parameters")
			}

			if elems[1].Typ != expr.ExprList && elems[1].Typ != expr.ExprNil {
				return fmt.Errorf("malformed 'code' form")
			}

			arglist := elems[1].List

			if elems[2].Typ != expr.ExprList && elems[2].Typ != expr.ExprNil {
				return fmt.Errorf("malformed 'code' form")
			}

			freevars := elems[2].List
			body := elems[3]

			// assign stack location for each argument
			for i, arg := range arglist {
				if arg.Typ != expr.ExprIdent {
					return fmt.Errorf("malformed 'code' form")
				}
				c.env[arg.Ident] = location{
					location: stack,
					offset:   -wordsize * (i + 1),
				}
			}

			// adjust the stack so that we point on top
			// of the arguments
			c.si = -wordsize * (len(arglist) + 1)

			// assign closure location for each free variable
			for i, arg := range freevars {
				if arg.Typ != expr.ExprIdent {
					return fmt.Errorf("malformed 'code' form")
				}
				c.env[arg.Ident] = location{
					location: closure,
					offset:   -wordsize * (i + 1),
				}
			}

			err := c.compileExpr(body)
			if err != nil {
				return fmt.Errorf("error compiling body in 'code' form: %w", err)
			}

			c.emit("ret")
			c.clearEnv()
			c.si = -wordsize
			return nil
		},

		"labelcall": func(c *Compiler, elems []expr.E) error {
			if len(elems) < 2 {
				return fmt.Errorf("labelcall form must contain at least 1 parameter")
			}

			if elems[1].Typ != expr.ExprIdent {
				return fmt.Errorf("malformed 'labelcall' form")
			}

			l := elems[1].Ident
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
			c.emit("addl $%d, %%esp", spSlot)
			c.emit("call %s", l)
			// restore esp
			c.emit("addl $%d, %%esp", -spSlot)
			c.si = siBefore

			return nil
		},

		"funcall": func(c *Compiler, elems []expr.E) error {
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
			c.emit("movl %%edi, %d(%%esp)", siBefore)

			err := c.compileExpr(f)
			if err != nil {
				return fmt.Errorf("error compiling function in funcall: %w", err)
			}

			// move new closure into closure pointer
			c.emit("movl %%eax, %%edi")
			// clear tag
			c.emit("andl $-8, %%edi")

			// handle call and return
			c.emit("movl 0(%%edi), %%ebx")
			c.emit("addl $%d, %%esp", spSlot)
			c.emit("call *%%ebx")
			c.emit("addl $%d, %%esp", -spSlot)
			c.si = siBefore

			return nil
		},

		"closure": func(c *Compiler, elems []expr.E) error {
			if len(elems) < 2 {
				return fmt.Errorf("closure form must contain at least 1 parameter")
			}

			if elems[1].Typ != expr.ExprIdent {
				return fmt.Errorf("malformed 'labelcall' form")
			}

			l := elems[1].Ident
			c.emit("movl $%s, 0(%%esi)", l)
			for i, freevar := range elems[2:] {
				// TODO copy free variable value directly instead
				// of moving to eax then to heap
				if freevar.Typ != expr.ExprIdent {
					return fmt.Errorf("malformed closure form")
				}
				err := c.compileExpr(freevar)
				if err != nil {
					return fmt.Errorf(
						"error compiling free variable in closure form at index %d: %w",
						i,
						err,
					)
				}

				c.emit("movl %%eax, %d(%%esi)", wordsize*(i+1))
			}

			length := len(elems) - 2

			c.emit("movl $%d, %%ebx", length)
			// eax = esi | 6
			c.emit("movl %%esi, %%eax")
			c.emit("orl $6, %%eax") // 6 = closure tag
			// align size to next object boundary
			c.emit("addl $11, %%ebx")
			c.emit("andl $-8, %%ebx")
			// advance alloc ptr
			c.emit("addl %%ebx, %%esi")
			return nil
		},
		"lambda": func(c *Compiler, elems []expr.E) error {
			// 'lambda' shoudln't show up in preprocessed code,
			// but we leave it here so that variable capture
			// analysis is performed correctly
			return fmt.Errorf("lambda is not implemented")
		},
		// built in functions
		"add1": func(c *Compiler, elems []expr.E) error {
			x := elems[1]
			err := c.compileExpr(x)
			if err != nil {
				return fmt.Errorf("error compiling '%s' application: %w", "add1", err)
			}
			c.emit("addl $4, %%eax")
			return nil
		},
		"integer->char": func(c *Compiler, elems []expr.E) error {
			x := elems[1]
			err := c.compileExpr(x)
			if err != nil {
				return fmt.Errorf("error compiling '%s' application: %w", "integer->char", err)
			}
			c.emit("sall $%d, %%eax", charShift-fixnumShift)
			c.emit("orl $0x%x, %%eax", charTag)
			return nil
		},
		"char->integer": func(c *Compiler, elems []expr.E) error {
			x := elems[1]
			err := c.compileExpr(x)
			if err != nil {
				return fmt.Errorf("error compiling '%s' application: %w", "char->integer", err)
			}
			c.emit("sarl $%d, %%eax", charShift-fixnumShift)
			return nil
		},

		"null?": func(c *Compiler, elems []expr.E) error {
			x := elems[1]
			err := c.compileExpr(x)
			if err != nil {
				return fmt.Errorf("error compiling '%s' application: %w", "zero?", err)
			}
			c.emit("cmpl $0x%x, %%eax", emptyList)
			c.emit("movl $0, %%eax")
			c.emit("sete %%al")
			c.emit("sall $7, %%eax")
			c.emit("orl $0x%x, %%eax", boolTag)

			return nil
		},
		"zero?": func(c *Compiler, elems []expr.E) error {
			x := elems[1]
			err := c.compileExpr(x)
			if err != nil {
				return fmt.Errorf("error compiling '%s' application: %w", "zero?", err)
			}
			c.emit("cmpl $0, %%eax")
			c.emit("movl $0, %%eax")
			c.emit("sete %%al")
			c.emit("sall $7, %%eax")
			c.emit("orl $0x%x, %%eax", boolTag)

			return nil
		},
		"+": func(c *Compiler, elems []expr.E) error {
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
			c.emit("addl %d(%%esp), %%eax", c.si)

			return nil
		},
		"-": func(c *Compiler, elems []expr.E) error {
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
			c.emit("subl %d(%%esp), %%eax", c.si)

			return nil
		},

		"cons": func(c *Compiler, elems []expr.E) error {
			if len(elems) != 3 {
				return fmt.Errorf("malformed cons expression")
			}
			x := elems[1]
			y := elems[2]

			err := c.compileExpr(x)
			if err != nil {
				return fmt.Errorf("error compiling cons expression: %w", err)
			}
			c.emit("movl %%eax, %d(%%esi)", 0*wordsize)

			err = c.compileExpr(y)
			if err != nil {
				return fmt.Errorf("error compiling cons expression: %w", err)
			}
			c.emit("movl %%eax, %d(%%esi)", 1*wordsize)

			c.emit("movl %%esi, %%eax")
			c.emit("orl $1, %%eax")

			c.emit("addl $%d, %%esi", 2*wordsize)

			return nil
		},

		"car": func(c *Compiler, elems []expr.E) error {
			if len(elems) != 2 {
				return fmt.Errorf("malformed car expression")
			}
			err := c.compileExpr(elems[1])
			if err != nil {
				return fmt.Errorf("error compiling car expression: %w", err)
			}

			c.emit("movl -1(%%eax), %%eax")
			return nil
		},

		"cdr": func(c *Compiler, elems []expr.E) error {
			if len(elems) != 2 {
				return fmt.Errorf("malformed cdr expression")
			}
			err := c.compileExpr(elems[1])
			if err != nil {
				return fmt.Errorf("error compiling car expression: %w", err)
			}

			c.emit("movl %d(%%eax), %%eax", wordsize-1)
			return nil
		},

		"make-vector": func(c *Compiler, elems []expr.E) error {
			if len(elems) != 2 {
				return fmt.Errorf("malformed make-vector expression")
			}

			err := c.compileExpr(elems[1])

			if err != nil {
				return fmt.Errorf("error compiling make-vector call: %w", err)
			}

			// set length
			c.emit("movl %%eax, 0(%%esi)")
			// save length
			c.emit("movl %%eax, %%ebx")
			// eax = esi | 2
			c.emit("movl %%esi, %%eax")
			c.emit("orl $2, %%eax") // 2 = vector tag
			// align size to next object boundary
			c.emit("addl $11, %%ebx")
			c.emit("andl $-8, %%ebx")
			// advance alloc ptr
			c.emit("addl %%ebx, %%esi")
			return nil
		},

		"vector-ref": func(c *Compiler, elems []expr.E) error {
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

			c.emit("addl $1, %%eax")
			c.emit("movl %d(%%esp), %%ebx", vectorIdx)
			c.emit("addl %%ebx, %%eax")
			c.emit("movl 0(%%eax), %%eax")

			return nil
		},

		"vector-set!": func(c *Compiler, elems []expr.E) error {
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
			c.emit("movl %d(%%esp), %%ebx", idxIdx)
			c.emit("addl $1, %%ebx")
			c.emit("addl %d(%%esp), %%ebx", vectorIdx)
			// move object into slot
			c.emit("movl %%eax, 0(%%ebx)")
			// set eax to vector pointer
			c.emit("movl %d(%%esp), %%eax", vectorIdx)

			return nil
		},
	}
}
