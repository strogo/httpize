package httpize

import (
	"fmt"
	"reflect"
)

// Exports is a map where keys are names of MethodProvider methods  
// and values are ParamDef. A Method will be called when a HTTP
// request where the last part of the URL.Path matches the key.
// Exported methods must have paramater types that match the returned types
// from ParamDef.CreateFunc and return (io.Reader, *httpize.Settings, error). If 
// Settings is nil, default httpize settings are used.
var exports = make(map[string]map[string][]string)

// Export will tell Handlers tied to a value whose type is named 
// typeName to call the method named methodName when the last part of URL.Path
// matches methodName. paramNames are names of URL parameters that will be 
// used to create arguments to the corresponding parameters of the method.
// Must be called before NewHandler.
func Export(typeName, methodName string, paramNames ...string) bool {
	if _, ok := exports[typeName]; !ok {
		exports[typeName] = make(map[string][]string)
	}
	exports[typeName][methodName] = paramNames
	return true
}

// CreateArgFromStringFunc is a function that will transform a string value of a
// URL parameter to an argument to be used in a method call.
type CreateArgFromStringFunc func(string) Arg

var types = make(map[string]CreateArgFromStringFunc)

// AddType allows a type named t to be use in parameters of exported methods.
// f must be a CreateArgFromStringFunc whose return value is assignable to the type
// named t.
func AddType(t string, f CreateArgFromStringFunc) bool {
	types[t] = f
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
			createFunc, ok := types[m.Type().In(i).Name()]
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
