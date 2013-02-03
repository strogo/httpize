package httpize

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type TestParamType string

func (d TestParamType) Check() error {
	if strings.Contains(string(d), "'") {
		return errors.New("TestParamType in wrong format")
	}
	return nil
}

func NewTestParamType(value string) TestParamType {
	return TestParamType(value)
}

func NewTestParamType2(a string) int {
	return 42
}

type TestApiProvider struct {
	settings *Settings
}

func (t *TestApiProvider) Httpize(methods Methods) {
	//    methods.Add("Echo", []ArgDef{ArgDef{"name", NewTestParamType}})
	methods.Add("Echo", []string{"name"}, []interface{}{NewTestParamType})
	methods.Add("Greeting", []string{}, []interface{}{})
	methods.Add("ThreeOhThree", []string{}, []interface{}{})
	methods.Add("BadEcho", []string{"name"}, []interface{}{NewTestParamType2})
}

func (t *TestApiProvider) Echo(name TestParamType) (io.Reader, *Settings, error) {
	return bytes.NewBufferString("Echo " + string(name)), t.settings, nil
}

func (t *TestApiProvider) Greeting() (io.Reader, *Settings, error) {
	return bytes.NewBufferString("Hello World"), t.settings, nil
}

func (t *TestApiProvider) ThreeOhThree() (io.Reader, *Settings, error) {
	err := Non500Error{303, "See Other", "http://lookhere"}
	return nil, t.settings, err
}

func (t *TestApiProvider) BadEcho(name TestParamType) (io.Reader, *Settings, error) {
	return bytes.NewBufferString("Echo " + string(name)), nil, nil
}

func checkCode(t *testing.T, r *httptest.ResponseRecorder, code int) {
	if r.Code != code {
		t.Fatalf("%d %v %s", r.Code, r.HeaderMap, r.Body)
	}
	t.Logf("%d %v %s", r.Code, r.HeaderMap, r.Body)
}

func TestTestApiProvider(t *testing.T) {
	var a TestApiProvider
	h := NewHandler(&a)

	recorder := httptest.NewRecorder()
	request, _ := http.NewRequest("GET", "http://host/Echo?name=Gopher", nil)
	h.ServeHTTP(recorder, request)
	checkCode(t, recorder, 200)
	if recorder.Body.String() != "Echo Gopher" {
		t.Fatal("incorrect response")
	}
	if v, ok := recorder.HeaderMap["Content-Type"]; !ok || v[0] != "text/html" {
		t.Fatalf("Content-Type header missing or invalid")
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

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequest("GET", "http://host/ThreeOhThree", nil)
	h.ServeHTTP(recorder, request)
	checkCode(t, recorder, 303)

	a.settings = new(Settings)

	a.settings.SetToDefault()
	a.settings.Cache = 300
	recorder = httptest.NewRecorder()
	request, _ = http.NewRequest("GET", "http://host/Greeting", nil)
	h.ServeHTTP(recorder, request)
	checkCode(t, recorder, 200)
	if _, ok := recorder.HeaderMap["Expires"]; !ok {
		t.Fatalf("Expires header missing")
	}
	now := time.Now()
	cacheTime, err := time.Parse(time.RFC1123, recorder.HeaderMap["Expires"][0])
	if err != nil || cacheTime.Before(now) {
		t.Fatalf("Expires header invalid")
	}

	a.settings.SetToDefault()
	a.settings.Gzip = true

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequest("GET", "http://host/Greeting", nil)
	request.Header.Add("Accept-Encoding", "gzip")
	h.ServeHTTP(recorder, request)
	checkCode(t, recorder, 200)
	if v, ok := recorder.HeaderMap["Content-Encoding"]; !ok || v[0] != "gzip" {
		t.Fatalf("Content-Encoding header missing or invalid")
	}

	a.settings.SetToDefault()

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequest("GET", "http://host/Greeting", nil)
	h.ServeHTTP(recorder, request)
	checkCode(t, recorder, 200)
	if _, ok := recorder.HeaderMap["Content-Encoding"]; ok {
		t.Fatalf("Unexpected Content-Encoding")
	}

	recorder = httptest.NewRecorder()
	request, _ = http.NewRequest("GET", "http://host/path/BadEcho?name=Gopher", nil)
	h.ServeHTTP(recorder, request)
	checkCode(t, recorder, 500)
}

type TestApiProviderPanic struct{}

func (t *TestApiProviderPanic) Httpize(methods Methods) {
	methods.Add("Echo", []string{"name"}, []interface{}{NewTestParamType})
}

func (t *TestApiProviderPanic) Echo(name TestParamType) (int, error) {
	return 42, nil
}

func TestTestApiProviderPanic(t *testing.T) {
	var a TestApiProviderPanic
	var err interface{} = nil
	func() {
		defer func() {
			err = recover()
		}()
		NewHandler(&a)
	}()
	if err == nil {
		t.Fatal("Panic expected but didn't happen.")
	}
	t.Logf("Panic happend %v", err)
}
