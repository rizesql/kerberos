package sdk

type Endpoint interface {
	Method() string
	Path() string
}
