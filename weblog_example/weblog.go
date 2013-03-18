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

var _ = httpize.AddType("*LogMessage", func(value string) httpize.Arg {
	return &LogMessage{time.Now(), value}
})

var webLog = make([]*LogMessage, 0)

type WebLogApi func(*bytes.Buffer, map[string]httpize.Arg) error

func (f WebLogApi) Call(args map[string]httpize.Arg) (io.WriterTo, *httpize.Settings, error) {
	buf := bytes.NewBufferString("")
	err := f(buf, args)
	return buf, nil, err
}

func Log(buf *bytes.Buffer, args map[string]httpize.Arg) error {
	msg := args["msg"].(*LogMessage)
	webLog = append(webLog, msg)
	return nil
}

var _ = httpize.Handle("/Log?msg *LogMessage", WebLogApi(Log))

func Read(buf *bytes.Buffer, args map[string]httpize.Arg) error {
	for _, msg := range webLog {
		_, err := buf.WriteString(msg.String())
		if err != nil {
			return err
		}
	}
	return nil
}

var _ = httpize.Handle("/Read", WebLogApi(Read))

func main() {
	http.ListenAndServe(":9001", nil)

	// Can now access the methods using:
	// http://localhost:9001/Log?msg=Hello World
	// http://localhost:9001/Read
}
