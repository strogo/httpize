package httpize

import (
	"log"
	"net/http"
	"regexp"
	"strings"
)

// for testing
var handlers = make(map[string]http.Handler)

// Add pattern to be handled. p: is a pattern to be handled. Patterns are like
// [path/]name([arguments]). If path/ is ommitted "/" is used. Arguments are
// a comma seprated list of two words. Where words are seperated by whitespace.
// First word is the key used to get a value from query part of the URL.
// The second word is a type registered with AddType. The patttern will match urls
// [path/]name?arg1_key=...&arg2_key=... etc. c is a Caller interface that
// will be called when pattern matches a given HTTP request. It will be
// passed arguments as specified by the pattern. Always returns true.
func Handle(p string, c Caller) bool {
	re, _ := regexp.Compile("^([^\\(]+)\\(([*,0-9,a-z,A-Z,_, ,\t]*)\\)$")
	parts := re.FindStringSubmatch(p)

	if len(parts) != 3 {
		log.Printf("httpize.Export handler pattern wrong. %s", p)
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
		re, _ = regexp.Compile("([0-9,a-z,A-Z,_]+)\\s+([*,0-9,a-z,A-Z,_]+)")
		paramParts := re.FindStringSubmatch(s)
		if len(paramParts) != 3 {
			log.Printf("httpize.Export handler pattern wrong. %s", p)
			return true
		}
		createFunc, ok := types[paramParts[2]]
		if !ok {
			log.Println(
				"httpize.Export: %s not a Httpize registered type",
				paramParts[2],
			)
		}

		a[i].key = paramParts[1]
		a[i].createFunc = createFunc
	}

	ds := new(Settings)
	ds.SetToDefault()

	handler := &handler{c, a, ds}
	http.Handle(path+"/"+name, handler)

	// for tests to access handler
	handlers[p] = handler

	return true
}

var types = make(map[string]func(string) Arg)

// Add type to be used in parameters of handled functions. t: name of type
// to be used in Handle() pattern. f: a function to create a new instance
// of the type, will be passed the string value of a URL parameter, type must implement
// Arg. Allways returns true.
func AddType(t string, f func(string) Arg) bool {
	types[t] = f
	return true
}

func GetHandlerForPattern(p string) http.Handler {
	return handlers[p]
}
