package httpize

import (
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Handler implements http.Handler
type Handler struct {
	calls           map[string]*caller
	defaultSettings *Settings
}

// Settings has options for handling HTTP request.
type Settings struct {
	Cache       int64
	ContentType string
	Gzip        bool
}

// SetToDefault sets: Cache = 0, Content-type = text/html, 
// gzip false.
func (s *Settings) SetToDefault() {
	s.Cache = 0
	s.ContentType = "text/html"
	s.Gzip = false
}

// NewHandler creates a Handler that serves requests to methods exported by
// a MethodProvider.
func NewHandler(provider MethodProvider) *Handler {
	h := new(Handler)

	if provider != nil {
		h.calls = buildCalls(provider)
	}

	h.defaultSettings = new(Settings)
	h.defaultSettings.SetToDefault()
	return h
}

// Non500Error is an error that can be returned by exported methods or an Arg 
// Check() method. Errors are considered 500 errors unless specifically of 
// this type.
type Non500Error struct {
	ErrorCode int
	ErrorStr  string
	Location  string
}

func (e Non500Error) Error() string {
	return e.ErrorStr
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
	call, ok := h.calls[methodName]
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

	foundArgs, err := call.buildArgs(func(s string) (string, bool) {
		v, ok := getParam[s]
		if !ok {
			return "", false
		}
		return v[0], true
	})

	if err != nil {
		if err == notArg {
			fiveHundredError(resp)
			log.Printf("Parameter in Method %s is not of type http.Arg", methodName)
			return
		} else {
			providerError(err, resp)
			return
		}
	}

	getParamCount := 0
	for _, v := range getParam {
		for i := 0; i < len(v); i++ {
			getParamCount++
		}
	}

	if len(foundArgs) != call.paramCount() || len(foundArgs) != getParamCount {
		fiveHundredError(resp)
		log.Printf("%s called incorrectly (URL: %s)", methodName, req.URL.String())
		return
	}

	reader, settings, err := call.call(foundArgs)

	if err != nil {
		providerError(err, resp)
		return
	}

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
