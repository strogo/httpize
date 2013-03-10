/*
Httpize is a web framework, that allows you use methods/types in web requests.

A value that has a method with a correct signature that can be exported so that,
the method will be called on the value for a HTTP request. This is achieved by:
Using AddType to add parameter types to be created by the handler from the URL of a HTTP request to be passed to a method.
Using Export to export methods that will be called by the the handler.
Using NewHandler to create a Handler for a value that has exported methods. The Handler is then passed
to http.Handle in the normal way.
*/
package httpize
