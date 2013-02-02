package httpize

type MethodProvider interface {
	Httpize(methods Methods)
}

type ParamType interface {
	Check() error
}

type ArgCreateFunc func(value string) ParamType

type Settings struct {
	Cache       int64
	ContentType string
	Gzip        bool
}

func (s *Settings) SetToDefault() {
	s.Cache = 0
	s.ContentType = "text/html"
	s.Gzip = false
}

// Handled errors are considered 500 errors unless specifically of type:
type Non500Error struct {
	ErrorCode int
	ErrorStr  string
	Location  string
}

func (e Non500Error) Error() string {
	return e.ErrorStr
}
