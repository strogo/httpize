package httpize

import (
	"errors"
	"io"
	"log"
	"reflect"
)

// Arg interface must be implemented by types that are used as parameters to
// exported methods. Arg.Check() is called on all arguments before calling an
// exported method, if it returns an error the call is not made.
type Arg interface {
	Check() error
}

type caller struct {
	methodFunc  reflect.Value
	argBuilders []argBuilder
}

type argBuilder struct {
	name       string
	createFunc reflect.Value
}

func (c *caller) paramCount() int {
	return len(c.argBuilders)
}

var notArg error = errors.New("Argument is not of type httpize.Arg")

func (c *caller) buildArgs(f func(s string) (string, bool)) ([]reflect.Value, error) {
	paramCount := c.paramCount()
	argValues := make([]reflect.Value, paramCount)

	found := 0
	for i := 0; i < paramCount; i++ {
		if v, ok := f(c.argBuilders[i].name); ok {
			var initString [1]reflect.Value
			initString[0] = reflect.ValueOf(v)
			argValues[i] = c.argBuilders[i].createFunc.Call(initString[:])[0]
			if arg, ok := argValues[i].Interface().(Arg); ok {
				err := arg.Check()
				if err != nil {
					return nil, err
				}
			} else {
				log.Printf(
					"Parameter %s not type httpize.Arg",
					c.argBuilders[i].name,
				)
				return nil, notArg
			}
			found++
		}
	}

	return argValues[:found], nil
}

func (c *caller) call(a []reflect.Value) (io.WriterTo, *Settings, error) {
	rvals := c.methodFunc.Call(a)

	// error can be not type error if nil for some reason
	if err, isError := rvals[2].Interface().(error); isError && err != nil {
		return nil, nil, err
	}
	settings := rvals[1].Interface().(*Settings)
	writerTo := rvals[0].Interface().(io.WriterTo)
	return writerTo, settings, nil
}
