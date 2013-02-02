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

type Handler struct {
	provider        MethodProvider
	methods         Methods
	defaultSettings *Settings
}

type ArgDef struct {
	name       string
	createFunc ArgCreateFunc
}

type Methods map[string][]ArgDef

func NewHandler(provider MethodProvider) *Handler {
	h := new(Handler)
	h.provider = provider
	h.methods = make(Methods)

	if h.provider != nil {
		h.provider.Httpize(h.methods)
	}

	for methodName := range h.methods {
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

func providerError(err error, resp http.ResponseWriter) {
	if e, ok := err.(Non500Error); ok {
		if e.ErrorCode == 301 || e.ErrorCode == 302 || e.ErrorCode == 303 {
			// might need to unset headers in here
			resp.Header().Set("Location", e.Location)
		}
		http.Error(resp, e.ErrorStr, e.ErrorCode)
	} else {
		fiveHundredError(resp)
		log.Print(err)
	}
}

func (h *Handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" && req.Method != "POST" {
		fiveHundredError(resp)
		log.Printf("Unsupported HTTP method: %s", req.Method)
		return
	}

	pathParts := strings.Split(req.URL.Path, "/")
	methodName := pathParts[len(pathParts)-1]
	argDefs, ok := h.methods[methodName]
	if !ok {
		fiveHundredError(resp)
		log.Printf("Method %s not defined (URL: %s)", methodName, req.URL.String())
		return
	}

	getParam, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		fiveHundredError(resp)
		log.Print(err)
		return
	}

	numArgs := len(argDefs)
	foundArgs := 0
	var argRval [10]reflect.Value
	for i := 0; i < numArgs; i++ {
		argName := argDefs[i].name
		if v, ok := getParam[argName]; ok {
			arg := argDefs[i].createFunc(v[0])
			err := arg.Check()
			if err != nil {
				providerError(err, resp)
				return
			}
			argRval[i] = reflect.ValueOf(arg)
			foundArgs++
		}
	}

	getParamCount := 0
	for _, v := range getParam {
		for i := 0; i < len(v); i++ {
			getParamCount++
		}
	}

	if foundArgs != numArgs || foundArgs != getParamCount {
		fiveHundredError(resp)
		log.Printf("%s called incorrectly (URL: %s)", methodName, req.URL.String())
		return
	}

	m := reflect.ValueOf(h.provider).MethodByName(methodName)
	rvals := m.Call(argRval[0:numArgs])

	// error can be not type error if nil for some reason
	if err, isError := rvals[2].Interface().(error); isError && err != nil {
		providerError(err, resp)
		return
	}

	settings := rvals[1].Interface().(*Settings)
	if settings == nil {
		settings = h.defaultSettings
	}

	if settings.ContentType != "" {
		resp.Header().Set("Content-Type", settings.ContentType)
	}

	if settings.Cache > 0 && req.Method == "GET" {
		t := time.Unix(time.Now().UTC().Unix()+settings.Cache, 0).UTC()
		resp.Header().Set("Expires", t.Format(time.RFC1123))
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

	reader := rvals[0].Interface().(io.Reader)
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
	numArgs := len(argNames)
	if numArgs != len(argCreateFuncs) {
		panic("Add method fail, argNames and argCreateFuncs array have different length")
	}
	if numArgs > 10 {
		panic("Add method fail, too many parameters (>10)")
	}
	a[methodName] = make([]ArgDef, numArgs)
	for i := 0; i < numArgs; i++ {
		a[methodName][i].name = argNames[i]
		a[methodName][i].createFunc = argCreateFuncs[i]
	}
}
