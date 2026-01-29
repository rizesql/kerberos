package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/rizesql/kerberos/internal/protocol"
)

type Kdc struct {
	root *Sdk
	cfg  Configuration
}

func newKdc(root *Sdk, cfg Configuration) *Kdc {
	return &Kdc{
		root: root,
		cfg:  cfg,
	}
}

func (kdc *Kdc) invoke(ctx context.Context, stub Endpoint, request any, response any) (err error) {
	path, err := url.JoinPath(kdc.cfg.ServerUrl, stub.Path())
	if err != nil {
		return fmt.Errorf("error generating URL: %w", err)
	}

	timeout := kdc.cfg.Timeout
	if timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	var body bytes.Buffer
	if err = json.NewEncoder(&body).Encode(request); err != nil {
		return fmt.Errorf("error encoding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, stub.Method(), path, &body)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	rawRes, err := kdc.root.cfg.Client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer func() {
		if closeErr := rawRes.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("error closing response body: %w", closeErr)
		}
	}()

	if rawRes.StatusCode != http.StatusOK {
		bodyBytes, readErr := io.ReadAll(rawRes.Body)
		if readErr != nil {
			return fmt.Errorf("received non-200 status code (%d) and failed to read body: %w", rawRes.StatusCode, readErr)
		}
		return fmt.Errorf("received non-200 status code (%d): %s", rawRes.StatusCode, string(bodyBytes))
	}

	if err = json.NewDecoder(rawRes.Body).Decode(response); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}

	return nil
}

func (kdc *Kdc) PostAS(ctx context.Context, req protocol.ASReq) (*protocol.ASRep, error) {
	var res protocol.ASRep
	if err := kdc.invoke(ctx, &protocol.ASEndpoint{}, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (kdc *Kdc) PostTGS(ctx context.Context, req protocol.TGSReq) (*protocol.TGSRep, error) {
	var res protocol.TGSRep
	if err := kdc.invoke(ctx, &protocol.TGSEndpoint{}, req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
