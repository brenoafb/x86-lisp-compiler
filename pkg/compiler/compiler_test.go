package compiler

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brenoafb/tinycompiler/pkg/parser"
)

func TestCompileExpr(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{
			code:     "42",
			expected: "movl $168, %eax\n",
		},
		{
			code: "(add1 42)",
			expected: `movl $168, %eax
addl $4, %eax
`,
		},
		{
			code: "(null? ())",
			expected: `movl $0x2f, %eax
cmpl $0x2f, %eax
movl $0, %eax
sete %al
sall $7, %eax
orl $0x1f, %eax
`,
		},
		{
			code: "(zero? 41)",
			expected: `movl $164, %eax
cmpl $0, %eax
movl $0, %eax
sete %al
sall $7, %eax
orl $0x1f, %eax
`,
		},
		{
			code: "(+ 13 87)",
			expected: `movl $348, %eax
movl %eax, -4(%esp)
movl $52, %eax
addl -4(%esp), %eax
`,
		},
		{
			code: "(let (x 1) x)",
			expected: `movl $4, %eax
movl %eax, -4(%esp)
movl -4(%esp), %eax
`,
		},
		{
			code: "(let (x 1) (y 2) (+ x y))",
			expected: `movl $4, %eax
movl %eax, -4(%esp)
movl $8, %eax
movl %eax, -8(%esp)
movl -8(%esp), %eax
movl %eax, -12(%esp)
movl -4(%esp), %eax
addl -12(%esp), %eax
`,
		},
		{
			code: "(if (zero? 1) 0 1)",
			expected: `movl $4, %eax
cmpl $0, %eax
movl $0, %eax
sete %al
sall $7, %eax
orl $0x1f, %eax
cmpl $0x1f, %eax
je L0
movl $0, %eax
jmp L1
L0:
movl $4, %eax
L1:
`,
		},
		{
			code: "(cons 1 2)",
			expected: `movl $4, %eax
movl %eax, 0(%esi)
movl $8, %eax
movl %eax, 4(%esi)
movl %esi, %eax
orl $1, %eax
addl $8, %esi
`,
		},
		{
			code: "(code () () (+ 1 2))",
			expected: `movl $8, %eax
movl %eax, -4(%esp)
movl $4, %eax
addl -4(%esp), %eax
ret
`,
		},
		{
			code: "(code (x) () (+ x 1))",
			expected: `movl $4, %eax
movl %eax, -8(%esp)
movl -4(%esp), %eax
addl -8(%esp), %eax
ret
`,
		},
		{
			code: "(code (x) (y) (+ x y))",
			expected: `movl 4(%edi), %eax
movl %eax, -8(%esp)
movl -4(%esp), %eax
addl -8(%esp), %eax
ret
`,
		},
		{
			code: "(closure f0)",
			expected: `movl $f0, 0(%esi)
movl $1, %ebx
movl %esi, %eax
orl $6, %eax
addl $11, %ebx
andl $-8, %ebx
addl %ebx, %esi
`,
		},
		{
			code: "(closure f0 4)",
			expected: `movl $f0, 0(%esi)
movl $16, %eax
movl %eax, 4(%esi)
movl $2, %ebx
movl %esi, %eax
orl $6, %eax
addl $11, %ebx
andl $-8, %ebx
addl %ebx, %esi
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			w := &bytes.Buffer{}
			c := NewCompiler(w)

			tokens, err := parser.Tokenize(tt.code)
			require.NoError(t, err)
			exprs, err := parser.Parse(tokens)
			require.NoError(t, err)
			require.Len(t, exprs, 1)
			expr := exprs[0]

			err = c.compileExpr(expr)
			require.NoError(t, err)
			require.Equal(t, tt.expected, w.String())
		})
	}
}

