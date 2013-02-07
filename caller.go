package httpize

import (
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
)

var notArg error = errors.New("Argument is not of type httpize.Arg")

type Caller struct {
	provider MethodProvider
	methods  Methods
}

type Methods map[string]*CallDef

type CallDef struct {
	methodFunc reflect.Value
	argDefs    []ArgDef
}

type ArgDef struct {
	name       string
	createFunc reflect.Value
}

func NewCaller(provider MethodProvider) *Caller {
	var c Caller
	c.provider = provider
	c.methods = make(Methods)

	if c.provider != nil {
		c.provider.Httpize(c.methods)
	}

	for methodName, callDef := range c.methods {
		v := reflect.ValueOf(c.provider)
		if v.Kind() == reflect.Invalid {
			panic("MethodProvider not valid")
		}
		m := v.MethodByName(methodName)
		if m.Kind() != reflect.Func {
			panic("Method not func")
		}
		if m.Type().NumOut() != 3 ||
			m.Type().Out(0).String() != "io.Reader" ||
			m.Type().Out(1).String() != "*httpize.Settings" ||
			m.Type().Out(2).String() != "error" {
			panic(fmt.Sprintf(
				"Method %s does not return (io.Reader, *httpize.Settings, error)",
				methodName,
			))
		}
		callDef.methodFunc = m
	}

	return &c
}

func (c *Caller) GetMethod(name string) *CallDef {
	callDef, ok := c.methods[name]
	if !ok {
		return nil
	}
	return callDef
}

func (m Methods) Add(methodName string, argNames []string, argCreateFuncs []interface{}) {
	numArgs := len(argNames)
	if numArgs != len(argCreateFuncs) {
		panic("Add method fail, argNames and argCreateFuncs array have different length")
	}
	if numArgs > 10 {
		panic("Add method fail, too many parameters (>10)")
	}

	callDef := new(CallDef)
	callDef.argDefs = make([]ArgDef, numArgs)
	for i := 0; i < numArgs; i++ {
		callDef.argDefs[i].name = argNames[i]

		v := reflect.ValueOf(argCreateFuncs[i])
		if v.Kind() != reflect.Func {
			panic("argCreateFunc is not a function")
		}
		if v.Type().NumIn() != 1 && v.Type().In(0).Kind() != reflect.String {
			panic("argCreateFunc incorrect parameter")
		}
		if v.Type().NumOut() != 1 {
			panic("argCreateFunc missing return value")
		}
		callDef.argDefs[i].createFunc = v
	}
	m[methodName] = callDef
}

func (c *CallDef) ArgCount() int {
	return len(c.argDefs)
}

func (c *CallDef) BuildArgs(f func(s string) (string, bool)) ([]reflect.Value, error) {
	var argReflect [10]reflect.Value

	found := 0
	numArgs := c.ArgCount()
	for i := 0; i < numArgs; i++ {
		if v, ok := f(c.argDefs[i].name); ok {
			var getValueReflect [1]reflect.Value
			getValueReflect[0] = reflect.ValueOf(v)
			argReflect[i] = c.argDefs[i].createFunc.Call(getValueReflect[:])[0]
			if arg, ok := argReflect[i].Interface().(Arg); ok {
				err := arg.Check()
				if err != nil {
					return nil, err
				}
			} else {
				log.Printf("Parameter %s not type httpize.Arg", c.argDefs[i].name)
				return nil, notArg
			}
			found++
		}
	}

	return argReflect[:found], nil
}

func (c *CallDef) Call(args []reflect.Value) (io.Reader, *Settings, error) {
	rvals := c.methodFunc.Call(args)

	// error can be not type error if nil for some reason
	if err, isError := rvals[2].Interface().(error); isError && err != nil {
		return nil, nil, err
	}
	settings := rvals[1].Interface().(*Settings)
	reader := rvals[0].Interface().(io.Reader)
	return reader, settings, nil
}
