# x86 Lisp Compiler

This is a compiler for a Lisp-like language that targets x86 (not x86_64). It's written in Go.

I used Abdulaziz Ghuloum's "An Incremental Approach to Compiler Construction" paper as a reference.


## Setting up

You need to have `zig` and `qemu` (if running on anything other than x86/i386) installed.

In Debian, Arch Linux and others, you can install `qemu-user-static`.

## Running

`lisp_entry.lisp` is used as an entry point.

```
(+ 1 2)
```

Run `make run` to compile and execute.

```
3
```

## Features:

- Let bindings

```
(let (x 1)
     (y 2)
  (+ x y))
```

- Lambdas

```
(let (f (lambda (x) (+ x 1)))
  (f 1))
```

- closures

```
(let (a 13)
  ((lambda (x) (+ x a)) 37))
```

- separate compilation

You can define a function in one file and reference it in another file.

```
(defun next (x) (+ x 1))
```

```
(next 1)
```

