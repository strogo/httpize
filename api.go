package httpize

type ApiProvider interface {
	Httpize(methods ApiMethods)
	GetHttpSettings() *Settings
}

type ArgType interface {
	Check() error
}

type NewArgFunc func(value string) ArgType

type Settings struct {
}
