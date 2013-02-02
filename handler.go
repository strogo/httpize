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
	provider        MethodProvider
	methods         Methods
	defaultSettings *Settings
}

type argDefs struct {
	name       []string
	createFunc []ArgCreateFunc
}

type Methods map[string]*argDefs

func NewHandler(provider MethodProvider) *HttpHandler {
	h := new(HttpHandler)
	h.provider = provider
	h.methods = make(Methods)

	if h.provider != nil {
		h.provider.Httpize(h.methods)
	}

	for methodName, _ := range h.methods {
		v := reflect.ValueOf(h.provider)
		if v.Kind() == reflect.Invalid {
			panic("MethodProvider not valid")
		}
		m := v.MethodByName(methodName)
		if m.Kind() != reflect.Func {
			panic("Method not func")
		}
		if m.Type().NumOut() != 3 ||
			m.Type().Out(0).String() != "io.Reader" ||
			m.Type().Out(1).String() != "*httpize.Settings" ||
			m.Type().Out(2).String() != "error" {
			panic(fmt.Sprintf(
				"Method %s does not return (io.Reader, *httpize.Settings, error)",
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
	argDef, ok := a.methods[methodName]
	if !ok {
		fiveHundredError(resp)
		log.Printf("Method %s not defined (URL: %s)", methodName, req.URL.String())
		return
	}
	numArgs := len(argDef.name)

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
	for i := 0; i < numArgs; i++ {
		argName := argDef.name[i]
		if _, ok := getParam[argName]; !ok {
			break
		}
		arg := argDef.createFunc[i](getParam[argName][0])
		err := arg.Check()
		if err != nil {
			non500Error, isNon500Error := err.(Non500Error)
			if isNon500Error {
				non500Error.write(resp)
			} else {
				fiveHundredError(resp)
				log.Printf("Method %s '%s' parameter error: %s", methodName, argName, err)
			}
			return
		}
		argRval[i] = reflect.ValueOf(arg)
		foundArgs++
	}

	if foundArgs != numArgs || foundArgs != getParamCount {
		fiveHundredError(resp)
		log.Printf(
			"Method %s(%s) called incorrectly (URL: %s)",
			methodName,
			strings.Join(argDef.name, ", "),
			req.URL.String(),
		)
		return
	}

	m := reflect.ValueOf(a.provider).MethodByName(methodName)
	r := m.Call(argRval[0:numArgs])

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

func (a Methods) Add(methodName string, argNames []string, argCreateFuncs []ArgCreateFunc) {
	numParams := len(argNames)
	if numParams != len(argCreateFuncs) {
		panic("Add method fail, paramNames and newArgFuncs array have different length")
	}
	if numParams > 10 {
		panic("Add method fail, too many parameters (>10)")
	}

	a[methodName] = &argDefs{argNames, argCreateFuncs}
}
