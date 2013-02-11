package httpize

import (
    "fmt"
    "reflect"
)

type Methods map[string]*caller

type ArgDef struct {
	Name       string
	CreateFunc interface{}
}

func (m Methods) Add(methodName string, argDefs []ArgDef) {
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
	m[methodName] = caller
}

func (m Methods) getProviderMethods(provider MethodProvider) {
	v := reflect.ValueOf(provider)
	if v.Kind() == reflect.Invalid {
		panic("MethodProvider not valid")
	}

	for methodName, caller := range m {
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
