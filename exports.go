package httpize

import (
	"fmt"
	"reflect"
)

// Used by MethodProvider.Httpize()
type Exports struct {
	methods map[string]*caller
}

// ArgDef defines a the Name of an argument and the CreateFunc that creates the
// argument from a string value.
// CreateFunc must be a func(string) and have a return a type that implements 
// httpize.Arg. 
type ArgDef struct {
	Name       string
	CreateFunc interface{}
}

// Add adds a method methodName that is called with arguments argDefs
// to be exported. methodName must be the name of a method defined in 
// the MethodProvider whose parameters match the []ArgDef and returns
// (io.Reader, *httpize.Settings, error). If Settings is nil, default httpize 
// settings are used.
func (e *Exports) Add(methodName string, argDefs []ArgDef) {
	numArgs := len(argDefs)
	if numArgs > 10 {
		panic("Add method fail, too many parameters (>10)")
	}

	caller := new(caller)
	caller.args = make([]args, numArgs)
	for i := 0; i < numArgs; i++ {
		caller.args[i].name = argDefs[i].Name

		v := reflect.ValueOf(argDefs[i].CreateFunc)
		if v.Kind() != reflect.Func {
			panic("argCreateFunc is not a function")
		}
		if v.Type().NumIn() != 1 && v.Type().In(0).Kind() != reflect.String {
			panic("argCreateFunc incorrect parameter")
		}
		if v.Type().NumOut() != 1 {
			panic("argCreateFunc missing return value")
		}
		caller.args[i].createFunc = v
	}
	e.methods[methodName] = caller
}

func (e *Exports) getProviderMethods(provider MethodProvider) {
	v := reflect.ValueOf(provider)
	if v.Kind() == reflect.Invalid {
		panic("MethodProvider not valid")
	}

	for methodName, caller := range e.methods {
		me := v.MethodByName(methodName)
		if me.Kind() != reflect.Func {
			panic("Method not func")
		}
		if me.Type().NumOut() != 3 ||
			me.Type().Out(0).String() != "io.Reader" ||
			me.Type().Out(1).String() != "*httpize.Settings" ||
			me.Type().Out(2).String() != "error" {
			panic(fmt.Sprintf(
				"Method %s does not return (io.Reader, *httpize.Settings, error)",
				methodName,
			))
		}
		caller.methodFunc = me
	}
}
