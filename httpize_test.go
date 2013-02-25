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

// A MethodProvider that exports an Echo Method
type SimpleMethodProvider struct{}

func (s *SimpleMethodProvider) Httpize() Exports {
	return Exports{
		"Echo": {"thing"},
	}
}

func (s *SimpleMethodProvider) Echo(thing SafeString) (io.WriterTo, *Settings, error) {
	return bytes.NewBufferString("Echo " + string(thing)), nil, nil
}

func TestSimpleMethodProvider(t *testing.T) {
	var s SimpleMethodProvider
	h := NewHandler(&s)

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "http://host/Echo?thing=Gopher", nil)
	h.ServeHTTP(recorder, request)
	if recorder.Body.String() != "Echo Gopher" {
		t.Fatal("incorrect response")
	}

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequest("GET", "http://host/Echo?name=Go'pher", nil)
	h.ServeHTTP(recorder, request)
	if recorder.Code != 500 {
		t.Fatalf("expect 500 error code, got: %d", recorder.Code)
	}
}
