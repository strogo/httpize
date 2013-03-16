package httpize

import (
	"io"
)

// Caller interface must be implemented by values to be handled. Will be called
// when HTTP request is handled. args: is array of Arg. The length and underlying
// types of elements is the same as specified by handler pattern. Arg.Check() has
// been called on each element. Return io.WriterTo will be used to write HTTP
// response body. Return *Settings is used to set HTTP options. If nil
// defaults as per Settings.SetToDefault() will be used. Return error if not
// nil causes HTTP 500 error responses, unless of is of type Non500Error in which
// the error code can be specified.
type Caller interface {
	Call(args []Arg) (io.WriterTo, *Settings, error)
}

// Arg.Check() is called on all arguments before calling an Caller.Call, 
// if it returns an error the call is not made and causes HTTP 500 error 
// response, unless of is of type Non500Error in which the error code can be 
// specified.
type Arg interface {
	Check() error
}

type argBuilderSlice []argBuilder

type argBuilder struct {
	key        string
	createFunc func(string) Arg
}

func (b argBuilderSlice) buildArgs(args []Arg, f func(s string) (string, bool)) (int, error) {
	paramCount := len(b)

	found := 0
	for i := 0; i < paramCount; i++ {
		if v, ok := f(b[i].key); ok {
			arg := b[i].createFunc(v)
			err := arg.Check()
			if err != nil {
				return found, err
			}
			args[i] = arg
			found++
		}
	}

	return found, nil
}
