package server

type Config struct {
	MaxReqBodySize int64
}

type Option func(*Config)

func WithMaxReqBodySize(size int64) Option {
	return func(c *Config) {
		c.MaxReqBodySize = size
	}
}
