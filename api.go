package httpize

type ApiProvider interface {
	Httpize(methods ApiMethods)
}

type ArgType interface {
	Check() error
}

type NewArgFunc func(value string) ArgType

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
