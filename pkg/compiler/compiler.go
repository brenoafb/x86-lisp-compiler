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

	expr, err = c.preprocess(expr)

	if err != nil {
		return fmt.Errorf("preprocessor error: %w", err)
	}

	c.preamble()
	c.compileExpr(expr)

	return nil
}

func (c *Compiler) gatherLambdas(
	expr interface{},
	counter *int,
	lambdas map[string]interface{},
) (interface{}, error) {
	switch expr.(type) {
	case []interface{}:
		elems := expr.([]interface{})
		if elems[0] == "lambda" {
			args := elems[1].([]interface{})
			freeVars := elems[2].([]interface{})
			body := elems[3]

			var err error
			body, err = c.gatherLambdas(body, counter, lambdas)
			if err != nil {
				return nil, fmt.Errorf("error gathering lambdas recursively: %w", err)
			}

			k := *counter
			*counter = *counter + 1
			label := fmt.Sprintf("f%d", k)

			newExpr := []interface{}{
				"closure",
				label,
			}

			for _, freeVar := range freeVars {
				newExpr = append(newExpr, freeVar)
			}

			code := []interface{}{
				"code",
				args,
				freeVars,
				body,
			}

			lambdas[label] = code
			return newExpr, nil
		}

		newExpr := make([]interface{}, 0, len(elems))

		for _, elem := range elems {
			elem, err := c.gatherLambdas(elem, counter, lambdas)
			if err != nil {
				return nil, fmt.Errorf("error annotating free variables in sub expression: %w", err)
			}
			newExpr = append(newExpr, elem)
		}

		return newExpr, nil

	default:
		return expr, nil
	}
}

func (c *Compiler) preprocess(expr interface{}) (interface{}, error) {
	expr, err := c.annotateFreeVariables(expr)

	if err != nil {
		return nil, fmt.Errorf("preprocess: error annotating lambdas: %w", err)
	}

	counter := 0
	lambdas := make(map[string]interface{})

	expr, err = c.gatherLambdas(expr, &counter, lambdas)

	if err != nil {
		return nil, fmt.Errorf("preprocess: error gathering lambdas: %w", err)
	}

	labels := []interface{}{}

	for k, v := range lambdas {
		labels = append(labels, []interface{}{
			k,
			v,
		})
	}

	result := []interface{}{
		"labels",
		labels,
		expr,
	}

	return result, nil
}

func (c *Compiler) annotateFreeVariables(
	expr interface{},
) (interface{}, error) {
	switch expr.(type) {
	case []interface{}:
		elems := expr.([]interface{})
		if len(elems) == 0 {
			return elems, nil
		}

		head := elems[0]

		if head == "lambda" {
			if len(elems) != 3 {
				return nil, fmt.Errorf("lambda form must contain 3 elements")
			}

			args := elems[1].([]interface{})
			body := elems[2]

			freeVars := make(map[string]struct{})
			argMap := make(map[string]struct{})
			for _, arg := range args {
				v := arg.(string)
				argMap[v] = struct{}{}
			}

			err := c.gatherFreeVariables(body, argMap, freeVars)

			if err != nil {
				return nil, fmt.Errorf("error annotating lambda expression: %w", err)
			}

			freeVarList := make([]interface{}, 0, len(freeVars))

			for k := range freeVars {
				freeVarList = append(freeVarList, k)
			}

			body, err = c.annotateFreeVariables(body)

			newExpr := []interface{}{
				"lambda",
				args,
				freeVarList,
				body,
			}

			return newExpr, nil
		}

		newExpr := make([]interface{}, 0, len(elems))

		for _, elem := range elems {
			elem, err := c.annotateFreeVariables(elem)
			if err != nil {
				return nil, fmt.Errorf("error annotating free variables in sub expression: %w", err)
			}
			newExpr = append(newExpr, elem)
		}

		return newExpr, nil
	default:
		return expr, nil
	}
}

func (c *Compiler) gatherFreeVariables(
	expr interface{},
	args map[string]struct{},
	freeVars map[string]struct{},
) error {
	switch expr.(type) {
	case string:
		v := expr.(string)
		if _, ok := builtins[v]; ok {
			return nil
		}
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
			c.emit("movl %d(%%esp), %%eax", loc.offset)
		case closure:
			c.emit("movl %d(%%edi), %%eax", loc.offset)
		case heap:
			c.emit("movl %d(%%esi), %%eax", loc.offset)
		}
		return nil
	case int:
		x := expr.(int)
		x <<= fixnumShift

		c.emit("movl $%d, %%eax", x)

		return nil

	case []interface{}:
		elems := expr.([]interface{})
		if len(elems) == 0 {
			c.emit("movl $0x%x, %%eax", emptyList)
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

func (c *Compiler) preamble() {
	preamble := `    .text
    .globl  scheme_entry
    .p2align    2
scheme_entry:
movl %%eax, %%esi`
	c.emit(preamble)
}

func (c *Compiler) genLabel() string {
	n := c.labelCounter
	c.labelCounter++
	return fmt.Sprintf("L%d", n)
}
