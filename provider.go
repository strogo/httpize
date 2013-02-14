package httpize

import (
	"fmt"
	"reflect"
)

// MethodProvider is implemented by types that want be able to export methods.
type MethodProvider interface {
	Httpize() Exports
}

// Exports is a map where keys are names of MethodProvider methods  
// and values are ParamDef. A Method will be called when a HTTP
// request where the last part of the URL.Path matches the key.
// Exported methods must have paramater types that match the returned types
// from ParamDef.CreateFunc and return (io.Reader, *httpize.Settings, error). If 
// Settings is nil, default httpize settings are used.
type Exports map[string][]ParamDef

// ParamDef defines a the Name of a parameter and the CreateFunc that creates the
// argument to be passed to the exported method from a string value obtained
// from a URL parameter with the same name.
// CreateFunc must be a func(string) and have a return a type that implements 
// httpize.Arg. 
type ParamDef struct {
	Name       string
	CreateFunc interface{}
}

func buildCalls(p MethodProvider) map[string]*caller {
	v := reflect.ValueOf(p)
	if v.Kind() == reflect.Invalid {
		panic("MethodProvider not valid")
	}

	calls := make(map[string]*caller)

	exports := p.Httpize()
	for exportName, argDefs := range exports {
		m := v.MethodByName(exportName)
		if m.Kind() != reflect.Func {
			panic("Method not func")
		}
		if m.Type().NumOut() != 3 ||
			m.Type().Out(0).String() != "io.Reader" ||
			m.Type().Out(1).String() != "*httpize.Settings" ||
			m.Type().Out(2).String() != "error" {
			panic(fmt.Sprintf(
				"Export %s does not return (io.Reader, *httpize.Settings, error)",
				exportName,
			))
		}

		a := make([]argBuilder, len(argDefs))
		for i := 0; i < len(argDefs); i++ {
			w := reflect.ValueOf(argDefs[i].CreateFunc)
			if w.Kind() != reflect.Func {
				panic("ArgDef.CreateFunc is not a function")
			}
			if w.Type().NumIn() != 1 && w.Type().In(0).Kind() != reflect.String {
				panic("ArgDef.CreateFunc incorrect parameter")
			}
			if w.Type().NumOut() != 1 {
				panic("ArgDef.CreateFunc missing return value")
			}
			a[i].name = argDefs[i].Name
			a[i].createFunc = w
		}

		calls[exportName] = &caller{m, a}
	}

	return calls
}
