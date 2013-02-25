package httpize

import (
	"fmt"
	"reflect"
)

// MethodProvider is implemented by types that want be able to export methods.
//type MethodProvider interface {
//	Httpize() Exports
//}

// Exports is a map where keys are names of MethodProvider methods  
// and values are ParamDef. A Method will be called when a HTTP
// request where the last part of the URL.Path matches the key.
// Exported methods must have paramater types that match the returned types
// from ParamDef.CreateFunc and return (io.Reader, *httpize.Settings, error). If 
// Settings is nil, default httpize settings are used.
var exports = make(map[string]map[string][]string)

func Export(typeName, methodName string, paramNames ...string) bool {
	if _, ok := exports[typeName]; !ok {
		exports[typeName] = make(map[string][]string)
	}
	exports[typeName][methodName] = paramNames
	return true
}

// ParamDef defines a the Name of a parameter and the CreateFunc that creates the
// argument to be passed to the exported method from a string value obtained
// from a URL parameter with the same name.
// CreateFunc must be a func(string) and have a return a type that implements 
// httpize.Arg. 
//type ParamDef struct {
//	Name       string
//	Type       string
//}

type CreateArgFromStringFunc func(string) Arg

var Types = make(map[string]CreateArgFromStringFunc)

func AddType(name string, f CreateArgFromStringFunc) bool {
	Types[name] = f
	return true
}

func buildCalls(provider interface{}) map[string]*caller {
	v := reflect.ValueOf(provider)
	if v.Kind() == reflect.Invalid {
		panic("MethodProvider not valid")
	}

	calls := make(map[string]*caller)

	providerName := v.Type().String()
	providerExports := exports[providerName]
	for exportName, paramNames := range providerExports {
		m := v.MethodByName(exportName)
		if m.Kind() == reflect.Invalid {
			panic("Cant find " + providerName + " " + exportName)
		}
		if m.Kind() != reflect.Func {
			panic("Method not func")
		}
		if m.Type().NumOut() != 3 ||
			m.Type().Out(0).String() != "io.WriterTo" ||
			m.Type().Out(1).String() != "*httpize.Settings" ||
			m.Type().Out(2).String() != "error" {
			panic(fmt.Sprintf(
				"Export %s does not return (io.WriterTo, *httpize.Settings, error)",
				exportName,
			))
		}
		if m.Type().NumIn() != len(paramNames) {
			panic(fmt.Sprintf("Incorrect parameter count for %s", exportName))
		}

		a := make([]argBuilder, len(paramNames))
		for i := 0; i < len(paramNames); i++ {
			createFunc, ok := Types[m.Type().In(i).Name()]
			if !ok {
				panic(m.Type().In(i).Name() + " not a Httpize registered type")
			}
			a[i].name = paramNames[i]
			a[i].createFunc = createFunc
		}

		calls[exportName] = &caller{m, a}
	}

	return calls
}
