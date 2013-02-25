package main

import (
	"bytes"
	"errors"
	"github.com/timob/httpize"
	"io"
	"net/http"
)

type LogMessage string

func (l LogMessage) Check() error {
	if l == "" {
		return errors.New("empty log message")
	}
	return nil
}

func (l LogMessage) String() string {
	return string(l) + "\n"
}

var _ = httpize.AddType("LogMessage", func(value string) httpize.Arg {
	return LogMessage(value)
})

type WebLog struct {
	messages []LogMessage
}

var _ = httpize.Export("*main.WebLog", "Log", "msg")

func (w *WebLog) Log(msg LogMessage) (io.WriterTo, *httpize.Settings, error) {
	w.messages = append(w.messages, msg)
	return bytes.NewBufferString(""), nil, nil
}

var _ = httpize.Export("*main.WebLog", "Read")

func (w *WebLog) Read() (io.WriterTo, *httpize.Settings, error) {
	buf := bytes.NewBufferString("")
	for _, msg := range w.messages {
		_, err := buf.WriteString(msg.String())
		if err != nil {
			return nil, nil, err
		}
	}

	return buf, nil, nil
}

func main() {
	var w WebLog
	http.Handle("/app/", httpize.NewHandler(&w))
	http.ListenAndServe(":9000", nil)

	// Can now access the methods using:
	// http://localhost:9000/app/Log?m=Hello World!
	// http://localhost:9000/app/Read
}
