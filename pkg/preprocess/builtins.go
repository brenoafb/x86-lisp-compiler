package preprocess

var names []string = []string{
	"progn",
	"define",
	"let",
	"if",
	"_main",
	"code",
	"labelcall",
	"funcall",
	"closure",
	"lambda",
	"string-ref",
	"string-init",
	"add1",
	"ccall",
	"integer->char",
	"char->integer",
	"null?",
	"zero?",
	"+",
	"-",
	"cons",
	"car",
	"cdr",
	"make-vector",
	"vector-ref",
	"vector-set!",
}

var builtins map[string]struct{}

func init() {
	builtins = make(map[string]struct{})
	for _, name := range names {
		builtins[name] = struct{}{}
	}
}
