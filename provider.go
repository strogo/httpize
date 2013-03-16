package httpize

import (
	"log"
	"net/http"
	"regexp"
	"strings"
)

// for testing
var handlers = make(map[string]http.Handler)

func Export(c Caller, d string) bool {
	re, _ := regexp.Compile("^([^\\(]+)\\(([0-9,a-z,A-Z,_, ,\t]*)\\)$")
	parts := re.FindStringSubmatch(d)

	if len(parts) != 3 {
		log.Println("httpize.Export handler pattern wrong")
		return true
	}
	pathParts := strings.Split(parts[1], "/")
	l := len(pathParts)
	path := strings.Join(pathParts[0:l-1], "/")
	name := pathParts[l-1]

	params := strings.Split(parts[2], ",")
	re, _ = regexp.Compile("^\\s*$")
	if re.MatchString(parts[2]) {
		params = []string{}
	}

	a := make([]argBuilder, len(params))
	for i, s := range params {
		re, _ = regexp.Compile("([0-9,a-z,A-Z,_]+)\\s+([0-9,a-z,A-Z,_]+)")
		paramParts := re.FindStringSubmatch(s)
		if len(paramParts) != 3 {
			log.Println("httpize.Export handler pattern wrong")
			return true
		}
		createFunc, ok := types[paramParts[2]]
		if !ok {
			log.Println("httpize.Export: " + paramParts[2] + " not a Httpize registered type")
		}

		a[i].key = paramParts[1]
		a[i].createFunc = createFunc
	}

	ds := new(Settings)
	ds.SetToDefault()

	handler := &handler{c, a, ds}
	http.Handle(path+"/"+name, handler)

	// for tests to access handler
	handlers[d] = handler

	return true
}

var types = make(map[string]func(string) Arg)

// Add type to be used in parameters of exported methods. t: name of a Go type
// to export, must include package prefix. f: a function to create a new instance
// of the type, will be passed a value of a URL parameter, type must implement
// Arg. Allways returns true.
func AddType(t string, f func(string) Arg) bool {
	types[t] = f
	return true
}
