package httpize

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

type HttpHandler struct {
	api             ApiProvider
	methods         ApiMethods
	defaultSettings *Settings
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
		if m.Type().NumOut() != 3 ||
			m.Type().Out(0).String() != "io.Reader" ||
			m.Type().Out(1).String() != "*httpize.Settings" ||
			m.Type().Out(2).String() != "error" {
			panic(fmt.Sprintf(
				"ApiMethod %s does not return (io.Reader, *httpize.Settings, error)",
				methodName,
			))
		}
	}

	h.defaultSettings = new(Settings)
	h.defaultSettings.SetToDefault()
	return h
}

func fiveHundredError(resp http.ResponseWriter) {
	http.Error(resp, "error", 500)
}

func (a *HttpHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" && req.Method != "POST" {
		fiveHundredError(resp)
		log.Printf("Unsupported HTTP method: %s", req.Method)
		return
	}

	pathParts := strings.Split(req.URL.Path, "/")
	methodName := pathParts[len(pathParts)-1]
	methodDef, ok := a.methods[methodName]
	if !ok {
		fiveHundredError(resp)
		log.Printf("Method %s not defined (URL: %s)", methodName, req.URL.String())
		return
	}
	numParams := len(methodDef.paramName)

	getParam, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		fiveHundredError(resp)
		log.Print(err)
		return
	}

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
			non500Error, isNon500Error := err.(Non500Error)
			if isNon500Error {
				non500Error.write(resp)
			} else {
				fiveHundredError(resp)
				log.Printf("Method %s '%s' parameter error: %s", methodName, paramName, err)
			}
			return
		}
		argRval[i] = reflect.ValueOf(arg)
		foundArgs++
	}

	if foundArgs != numParams || foundArgs != getParamCount {
		fiveHundredError(resp)
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

	// error can be not type error if nil for some reason
	if err, isError := r[2].Interface().(error); isError && err != nil {
		non500Error, isNon500Error := err.(Non500Error)
		if isNon500Error {
			non500Error.write(resp)
		} else {
			fiveHundredError(resp)
			log.Print(err)
		}
		return
	}

	settings := r[1].Interface().(*Settings)
	if settings == nil {
		settings = a.defaultSettings
	}

	if settings.ContentType != "" {
		resp.Header().Set("Content-Type", settings.ContentType)
	}

	if settings.Cache > 0 && req.Method == "GET" {
		var a time.Time
		a = time.Unix(time.Now().UTC().Unix()+settings.Cache, 0).UTC()
		resp.Header().Set("Expires", a.Format(time.RFC1123))
	}

	var compress io.Writer
	if settings.Gzip && strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
		resp.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(resp)
		compress = gz
		defer gz.Close()
	} else {
		compress = resp
	}

	reader := r[0].Interface().(io.Reader)
	if reader == nil {
		fiveHundredError(resp)
		log.Printf("Method %s returned nil reader and error", methodName)
		return
	}

	_, err = io.Copy(compress, reader)
	if err != nil {
		fiveHundredError(resp)
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
