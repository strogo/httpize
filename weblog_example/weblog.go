package main

import (
	"bytes"
	"errors"
	"github.com/timob/httpize"
	"io"
	"net/http"
	"time"
)

type LogMessage struct {
	timestamp time.Time
	msg       string
}

func (l *LogMessage) Check() error {
	if l.msg == "" {
		return errors.New("empty log message")
	}
	return nil
}

func (l *LogMessage) String() string {
	return l.timestamp.Format(time.UnixDate) + ": " + string(l.msg) + "\n"
}

var _ = httpize.AddType("*main.LogMessage", func(value string) httpize.Arg {
	return &LogMessage{time.Now(), value}
})

type WebLog struct {
	messages []*LogMessage
}

var _ = httpize.Export((*WebLog).Log, "Log", "msg")

func (w *WebLog) Log(msg *LogMessage) (io.WriterTo, *httpize.Settings, error) {
	w.messages = append(w.messages, msg)
	return bytes.NewBufferString(""), nil, nil
}

var _ = httpize.Export((*WebLog).Read, "Read")

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
	http.ListenAndServe(":9001", nil)

	// Can now access the methods using:
	// http://localhost:9001/app/Log?msg=Hello World!
	// http://localhost:9001/app/Read
}
