package httpize

import (
	"bufio"
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

	h.api.Httpize(h.methods)
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

	getParamCount := 0
	foundArgs := 0
	var argRval [10]reflect.Value
	for i := 0; i < len(methodDef.paramName); i++ {
		paramName := methodDef.paramName[i]
		if _, ok := getParam[paramName]; ok {
			arg := methodDef.newArgFunc[i](getParam[paramName][0])
			err := arg.Check()
			if err != nil {
				FiveHundredError(resp)
				log.Printf(
					"Method %s, %s argument error %s",
					methodName,
					paramName,
					err,
				)
				return
			}
			argRval[i] = reflect.ValueOf(arg)
			foundArgs++
		}
		getParamCount += len(getParam[paramName])
	}

	if getParamCount != len(methodDef.paramName) || foundArgs != len(methodDef.paramName) {
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
	r := m.Call(argRval[0:foundArgs])

	reader := r[0].Interface().(io.Reader)
	err = r[1].Interface().(error)

	if err != nil {
		FiveHundredError(resp)
		log.Print(err)
		return
	}

	_, err = io.Copy(resp, bufio.NewReader(reader))
	if err != nil {
		FiveHundredError(resp)
		log.Print(err)
	}
}

func (a ApiMethods) Add(methodName string, paramNames []string, newArgFuncs []NewArgFunc) {
	if len(paramNames) != len(newArgFuncs) {
		//panic
	}
	if len(newArgFuncs) > 10 {
		//panic
	}
	a[methodName] = &apiMethod{paramNames, newArgFuncs}
}
