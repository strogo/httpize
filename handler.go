package httpize

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

type HttpHandler struct {
	api      ApiProvider
	methods  ApiMethods
	settings *Settings
}

type apiMethod struct {
	paramName  []string
	newArgFunc []NewArgFunc
}

type ApiMethods map[string]*apiMethod

func NewHandler(api ApiProvider) *HttpHandler {
	h := new(HttpHandler)
	h.api = api
	h.methods = make(ApiMethods)

	if h.api != nil {
		h.api.Httpize(h.methods)
	}

	for methodName, _ := range h.methods {
		v := reflect.ValueOf(h.api)
		if v.Kind() == reflect.Invalid {
			panic("ApiProvider not valid")
		}
		m := v.MethodByName(methodName)
		if m.Kind() != reflect.Func {
			panic("ApiMethod not func")
		}
		if m.Type().NumOut() != 2 ||
			m.Type().Out(0).Name() != "Reader" ||
			m.Type().Out(1).Name() != "error" {
			panic("ApiMethod does not return (io.Reader, error)")
		}
	}

	return h
}

func FiveHundredError(resp http.ResponseWriter) {
	http.Error(resp, "error", 500)
}

func (a *HttpHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		FiveHundredError(resp)
		log.Printf("Unsupported HTTP method: %s", req.Method)
		return
	}

	getParam, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		FiveHundredError(resp)
		log.Print(err)
		return
	}

	pathParts := strings.Split(req.URL.Path, "/")
	methodName := pathParts[len(pathParts)-1]
	methodDef, ok := a.methods[methodName]
	if !ok {
		FiveHundredError(resp)
		log.Printf("Method %s not defined (URL: %s)", methodName, req.URL.String())
		return
	}
	numParams := len(methodDef.paramName)

	getParamCount := 0
	for _, v := range getParam {
		for i := 0; i < len(v); i++ {
			getParamCount++
		}
	}

	foundArgs := 0
	var argRval [10]reflect.Value
	for i := 0; i < numParams; i++ {
		paramName := methodDef.paramName[i]
		if _, ok := getParam[paramName]; !ok {
			break
		}
		arg := methodDef.newArgFunc[i](getParam[paramName][0])
		err := arg.Check()
		if err != nil {
			FiveHundredError(resp)
			log.Printf("Method %s '%s' parameter error: %s", methodName, paramName, err)
			return
		}
		argRval[i] = reflect.ValueOf(arg)
		foundArgs++
	}

	if foundArgs != numParams || foundArgs != getParamCount {
		FiveHundredError(resp)
		log.Printf(
			"Method %s(%s) called incorrectly (URL: %s)",
			methodName,
			strings.Join(methodDef.paramName, ", "),
			req.URL.String(),
		)
		return
	}

	m := reflect.ValueOf(a.api).MethodByName(methodName)
	r := m.Call(argRval[0:numParams])

	reader := r[0].Interface().(io.Reader)
	errVal := r[1].Interface()

	if errVal != nil {
		FiveHundredError(resp)
		log.Print(errVal.(error))
		return
	}

	_, err = io.Copy(resp, reader)
	if err != nil {
		FiveHundredError(resp)
		log.Print(err)
	}
}

func (a ApiMethods) Add(methodName string, paramNames []string, newArgFuncs []NewArgFunc) {
	if len(paramNames) != len(newArgFuncs) {
		panic("Add method fail, paramNames and newArgFuncs array have different length")
	}
	if len(newArgFuncs) > 10 {
		panic("Add method fail, too many parameters (>10)")
	}

	a[methodName] = &apiMethod{paramNames, newArgFuncs}
}
