package httpize

import (
    "net/http"
    "errors"
    "bytes"
    "io"
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

func (w *WebLog) Log(m LogMessage) (io.Reader, *Settings, error) {
    w.messages = append(w.messages, m)
    return bytes.NewBufferString(""), nil, nil
}

func (w *WebLog) Read() (io.Reader, *Settings, error) {
    buf := bytes.NewBufferString("")
    for i := 0; i < len(w.messages); i++ {
        _, err := buf.WriteString(w.messages[i].String() + "\n")
        if err != nil {
            return nil, nil, err
        }
    }
    
    return buf, nil, nil
}

func (w *WebLog) Httpize(methods Methods) {
    methods.Add("Log", []ArgDef{{"m", NewLogMessage}})
    methods.Add("Read", []ArgDef{})
}

func ExampleWebLog() {
    var w WebLog
    http.Handle("/", NewHandler(&w))
    http.ListenAndServe(":9000", nil)    
}
