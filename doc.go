/*
Httpize is a web framework, that allows you use methods/types in web requests.

A value that has a method with a correct signature can be exported so that,
the method will be called on the value for a HTTP request. This is achieved by:
Adding types to be created by the handler from HTTP requests using AddType.
Exporting methods that will be called by the the handler using Export.
Creating handlers for a value that has exported methods. Handlers are then passed
to http.Handle in the normal way.
*/
package httpize
