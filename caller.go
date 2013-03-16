package httpize

import (
	"io"
)

type Caller interface {
	Call(args []Arg) (io.WriterTo, *Settings, error)
}

// Arg interface must be implemented by types that are used as parameters to
// exported methods. Arg.Check() is called on all arguments before calling an
// exported method, if it returns an error the call is not made.
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
