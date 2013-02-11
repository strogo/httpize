package httpize

import (
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
)

var notArg error = errors.New("Argument is not of type httpize.Arg")

type Methods map[string]*Caller

type Caller struct {
	methodFunc reflect.Value
	args       []Args
}

type Args struct {
	name       string
	createFunc reflect.Value
}

func NewMethods(provider MethodProvider) Methods {
	m := make(Methods)

	if provider != nil {
		provider.Httpize(m)
	}

	for methodName, caller := range m {
		v := reflect.ValueOf(provider)
		if v.Kind() == reflect.Invalid {
			panic("MethodProvider not valid")
		}
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

	return m
}

func (m Methods) GetCaller(name string) *Caller {
	caller, ok := m[name]
	if !ok {
		return nil
	}
	return caller
}

func (m Methods) Add(methodName string, args []ArgDef) {
	numArgs := len(args)
	if numArgs > 10 {
		panic("Add method fail, too many parameters (>10)")
	}

	caller := new(Caller)
	caller.args = make([]Args, numArgs)
	for i := 0; i < numArgs; i++ {
		caller.args[i].name = args[i].Name

		v := reflect.ValueOf(args[i].CreateFunc)
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

func (c *Caller) ArgCount() int {
	return len(c.args)
}

func (c *Caller) BuildArgs(f func(s string) (string, bool)) ([]reflect.Value, error) {
	var argReflect [10]reflect.Value

	found := 0
	numArgs := c.ArgCount()
	for i := 0; i < numArgs; i++ {
		if v, ok := f(c.args[i].name); ok {
			var getValueReflect [1]reflect.Value
			getValueReflect[0] = reflect.ValueOf(v)
			argReflect[i] = c.args[i].createFunc.Call(getValueReflect[:])[0]
			if arg, ok := argReflect[i].Interface().(Arg); ok {
				err := arg.Check()
				if err != nil {
					return nil, err
				}
			} else {
				log.Printf("Parameter %s not type httpize.Arg", c.args[i].name)
				return nil, notArg
			}
			found++
		}
	}

	return argReflect[:found], nil
}

func (c *Caller) Call(args []reflect.Value) (io.Reader, *Settings, error) {
	rvals := c.methodFunc.Call(args)

	// error can be not type error if nil for some reason
	if err, isError := rvals[2].Interface().(error); isError && err != nil {
		return nil, nil, err
	}
	settings := rvals[1].Interface().(*Settings)
	reader := rvals[0].Interface().(io.Reader)
	return reader, settings, nil
}
