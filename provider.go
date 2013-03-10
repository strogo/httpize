package httpize

import (
	"fmt"
	"reflect"
)

var calls = make(map[string]*caller)

// Export method to be called by Handler. m: method to be called, must return
// (io.WriterTo, *httpize.Settings, error), paramers types must be registered
// with AddType. e: string to be matched to last part of URL.Path. p: URL
// parameters used create arguments to the corresponding parameters of the method.
// Must be called before NewHandler. Always returns true.
func Export(m interface{}, e string, p ...string) bool {
	mv := reflect.ValueOf(m)

	if mv.Kind() != reflect.Func || mv.Type().NumIn() == 0 {
		panic(fmt.Sprintf("Export is not method (%s)", mv.String()))
	}

	if mv.Type().NumOut() != 3 ||
		mv.Type().Out(0).String() != "io.WriterTo" ||
		mv.Type().Out(1).String() != "*httpize.Settings" ||
		mv.Type().Out(2).String() != "error" {
		panic(fmt.Sprintf(
			"Export %s does not return (io.WriterTo, *httpize.Settings, error)",
			mv.String(),
		))
	}

	if mv.Type().NumIn()-1 != len(p) {
		panic(fmt.Sprintf("Incorrect parameter count for %s", e))
	}

	a := make([]argBuilder, len(p))
	for i := 0; i < len(p); i++ {
		createFunc, ok := types[mv.Type().In(i+1).String()]
		if !ok {
			panic(mv.Type().In(i+1).String() + " not a Httpize registered type")
		}
		a[i].key = p[i]
		a[i].createFunc = createFunc
	}

	calls[mv.Type().In(0).String()+"-"+e] = &caller{mv, a}

	return true
}

var types = make(map[string]func(string) Arg)

// Add type to be used in parameters of exported methods. t: name of a Go type
// to export, must include package prefix. f: a function to create a new instance
// of the type, will be passed a value of a URL parameter, type must implement
// Arg. Allways returns true.
func AddType(t string, f func(string) Arg) bool {
	types[t] = f
	return true
}
