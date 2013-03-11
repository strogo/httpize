/*
Httpize is a web framework, that allows you use methods/types in web requests.

A method can be exported so the method will be called by a handler for a HTTP request. 
This is achieved by:

1) Using AddType() to add value creating functions for a type, so values for typed arguments to methods can 
be created from request URL parameters.

2) Using Export() to export methods that will be called by the the handler.

3) Using NewHandler() to create a Handler for a value that has exported methods.

The Handler is then passed to http.Handle in the normal way.
*/
package httpize
