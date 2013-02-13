package main

import (
	"bytes"
	"errors"
	"httpize"
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
	return string(l)
}

func NewLogMessage(init string) LogMessage {
	return LogMessage(init)
}

type WebLog struct {
	messages []LogMessage
}

func (w *WebLog) Log(m LogMessage) (io.Reader, *httpize.Settings, error) {
	w.messages = append(w.messages, m)
	return bytes.NewBufferString(""), nil, nil
}

func (w *WebLog) Read() (io.Reader, *httpize.Settings, error) {
	buf := bytes.NewBufferString("")
	for i := 0; i < len(w.messages); i++ {
		_, err := buf.WriteString(w.messages[i].String() + "\n")
		if err != nil {
			return nil, nil, err
		}
	}

	return buf, nil, nil
}

func (w *WebLog) Httpize() httpize.Exports {
	return httpize.Exports{
		"Log":  {{"m", NewLogMessage}},
		"Read": {},
	}
}

func main() {
	var w WebLog
	http.Handle("/", httpize.NewHandler(&w))
	http.ListenAndServe(":9000", nil)
}
