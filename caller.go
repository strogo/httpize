package httpize

import (
	"errors"
	"io"
	"log"
	"reflect"
)

type caller struct {
	methodFunc reflect.Value
	args       []args
}

type args struct {
	name       string
	createFunc reflect.Value
}

func (c *caller) argCount() int {
	return len(c.args)
}

type Arg interface {
	Check() error
}

var notArg error = errors.New("Argument is not of type httpize.Arg")

func (c *caller) buildArgs(f func(s string) (string, bool)) ([]reflect.Value, error) {
	var argReflect [10]reflect.Value

	found := 0
	numArgs := c.argCount()
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

func (c *caller) call(args []reflect.Value) (io.Reader, *Settings, error) {
	rvals := c.methodFunc.Call(args)

	// error can be not type error if nil for some reason
	if err, isError := rvals[2].Interface().(error); isError && err != nil {
		return nil, nil, err
	}
	settings := rvals[1].Interface().(*Settings)
	reader := rvals[0].Interface().(io.Reader)
	return reader, settings, nil
}
