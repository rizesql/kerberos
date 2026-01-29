package sdk

import (
	"net/http"
	"time"
)

const timeout = 60 * time.Second

type Sdk struct {
	Kdc *Kdc
	cfg Configuration
}

func New(opts ...SdkOption) *Sdk {
	sdk := &Sdk{
		cfg: Configuration{},
	}

	for _, opt := range opts {
		opt(sdk)
	}

	if sdk.cfg.Client == nil {
		sdk.cfg.Client = &http.Client{Timeout: timeout}
	}

	sdk.Kdc = newKdc(sdk, sdk.cfg)

	return sdk
}
