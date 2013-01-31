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

type TestArgType string

func (d TestArgType) Check() error {
	if strings.Contains(string(d), "'") {
		return errors.New("TestArgType in wrong format")
	}
	return nil
}

func NewTestArgType(value string) ArgType {
	return TestArgType(value)
}

type TestApiProvider struct{}

func (t *TestApiProvider) Httpize(methods ApiMethods) {
	methods.Add("Echo", []string{"name"}, []NewArgFunc{NewTestArgType})
	methods.Add("Greeting", []string{}, []NewArgFunc{})
}

func (t *TestApiProvider) GetHttpSettings() *Settings {
	return nil
}

func (t *TestApiProvider) Echo(name TestArgType) (io.Reader, error) {
	return bytes.NewBufferString("Echo " + string(name)), nil
}

func (t *TestApiProvider) Greeting() (io.Reader, error) {
	return bytes.NewBufferString("Hello World"), nil
}

func checkCode(t *testing.T, r *httptest.ResponseRecorder, code int) {
	if r.Code != code {
		t.Fatalf("%d %v %s", r.Code, r.HeaderMap, r.Body)
	}
	t.Logf("%d %v %s", r.Code, r.HeaderMap, r.Body)
}

func TestHttpize(t *testing.T) {
	var a TestApiProvider
	h := NewHandler(&a)

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "http://host/Echo?name=Gopher", nil)
	h.ServeHTTP(recorder, request)
	checkCode(t, recorder, 200)
	if recorder.Body.String() != "Echo Gopher" {
		t.Fatal("incorrect response")
	}

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequest("GET", "http://host/path/Echo?name=Gopher", nil)
	h.ServeHTTP(recorder, request)
	checkCode(t, recorder, 200)

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequest("GET", "http://host/", nil)
	h.ServeHTTP(recorder, request)
	checkCode(t, recorder, 500)

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequest("GET", "http://host", nil)
	h.ServeHTTP(recorder, request)
	checkCode(t, recorder, 500)

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequest("GET", "http://host/Nothere?name=Gopher", nil)
	h.ServeHTTP(recorder, request)
	checkCode(t, recorder, 500)

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequest("GET", "http://host/Echo?badparam=Gopher", nil)
	h.ServeHTTP(recorder, request)
	checkCode(t, recorder, 500)

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequest("GET", "http://host/Echo?name=Go'pher", nil)
	h.ServeHTTP(recorder, request)
	checkCode(t, recorder, 500)

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequest("GET", "http://host/Echo?name=Gopher&name=Gopher", nil)
	h.ServeHTTP(recorder, request)
	checkCode(t, recorder, 500)

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequest("GET", "http://host/Echo?name=Gopher&badparam=Gopher", nil)
	h.ServeHTTP(recorder, request)
	checkCode(t, recorder, 500)

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequest("GET", "http://host/Greeting", nil)
	h.ServeHTTP(recorder, request)
	checkCode(t, recorder, 200)

}
