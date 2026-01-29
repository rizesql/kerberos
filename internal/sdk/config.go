package sdk

import (
	"net/http"
	"time"
)

type Configuration struct {
	Client    *http.Client
	ServerUrl string
	Timeout   *time.Duration
}

type SdkOption func(*Sdk)

func WithClient(client *http.Client) SdkOption {
	return func(sdk *Sdk) {
		sdk.cfg.Client = client
	}
}

func WithServerUrl(serverUrl string) SdkOption {
	return func(sdk *Sdk) {
		sdk.cfg.ServerUrl = serverUrl
	}
}

func WithTimeout(timeout time.Duration) SdkOption {
	return func(sdk *Sdk) {
		sdk.cfg.Timeout = &timeout
	}
}
