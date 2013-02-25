package httpize

import (
	"fmt"
	"reflect"
)

var exports = make(map[string]map[string][]string)

// Export will tell Handlers created with a value whose type is named 
// t to call the method named m when the last part of URL.Path
// matches m. p are names of URL parameters that will be 
// used to create arguments to the corresponding parameters of the method.
// Methods must return (io.WriterTo, *httpize.Settings, error).
// Must be called before NewHandler. t must include package prefix.
// Always returns true.
func Export(t, m string, p ...string) bool {
	if _, ok := exports[t]; !ok {
		exports[t] = make(map[string][]string)
	}
	exports[t][m] = p
	return true
}

type createArgFromStringFunc func(string) Arg

var types = make(map[string]createArgFromStringFunc)

// AddType allows a type named t to be use in parameters of exported methods.
// f must be a function whose return value is assignable to the type
// named t and implements Arg. t must include package prefix. Always returns true.
func AddType(t string, f func(string) Arg) bool {
	types[t] = createArgFromStringFunc(f)
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
			createFunc, ok := types[m.Type().In(i).String()]
			if !ok {
				panic(m.Type().In(i).String() + " not a Httpize registered type")
			}
			a[i].name = paramNames[i]
			a[i].createFunc = createFunc
		}

		calls[exportName] = &caller{m, a}
	}

	return calls
}
