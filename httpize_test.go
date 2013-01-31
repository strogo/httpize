package httpize

import (
	"errors"
	"reflect"
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
}

func (t *TestApiProvider) GetHttpSettings() *Settings {
	return nil
}

func (t *TestApiProvider) Echo(name TestArgType) string {
	return "Echo " + string(name)
}

func (a *HttpHandler) Debug(t *testing.T, methodName string, argString string) {
	d := a.methods[methodName]
	m := reflect.ValueOf(a.api).MethodByName(methodName)

	var argRval [1]reflect.Value
	arg := d.newArgFunc[0](argString)
	err := arg.Check()
	if err != nil {
		t.Log(err)
		return
	}
	argRval[0] = reflect.ValueOf(arg)
	r := m.Call(argRval[:])

	rstr := r[0].Interface().(string)
	t.Log(rstr)
}

func TestHttpize(t *testing.T) {
	var a TestApiProvider

	h := NewHandler(&a)

	h.Debug(t, "Echo", "Tim O'Brien")
}

/*
func (b *blah) hello(dbname DbStringType) {
    conn := connect(dbname....)
}
*/
