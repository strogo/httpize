package httpize

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A type that will be used as httpize.Arg
type SafeString string

func (d SafeString) Check() error {
	if strings.Contains(string(d), "'") || d == "" {
		return errors.New("SafeString in wrong format")
	}
	return nil
}

var _ = AddType("SafeString", func(value string) Arg {
	return SafeString(value)
})

// A type that will be used as a httpize.Caller
type SimpleFunc func(map[string]Arg) string

func (f SimpleFunc) Call(args map[string]Arg) (io.WriterTo, *Settings, error) {
	return bytes.NewBufferString(f(args)), nil, nil
}

var _ = Handle("/Greet(thing SafeString)", SimpleFunc(Greet))

func Greet(args map[string]Arg) string {
	return "Hello " + string(args["thing"].(SafeString))
}

func TestSimpleFunc(t *testing.T) {

	h := GetHandlerForPattern("/Greet(thing SafeString)")

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "http://host/Greet?thing=Gopher", nil)
	h.ServeHTTP(recorder, request)
	if recorder.Body.String() != "Hello Gopher" {
		t.Fatal("incorrect response")
	}

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequest("GET", "http://host/Greet?thing=Go'pher", nil)
	h.ServeHTTP(recorder, request)
	if recorder.Code != 500 {
		t.Fatalf("expect 500 error code, got: %d", recorder.Code)
	}
}
