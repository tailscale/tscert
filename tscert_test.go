// Copyright (c) 2022 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tscert

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func fakeResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     strconv.Itoa(status) + " " + http.StatusText(status),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestSendUnexpectedStatus(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{"json error", 500, `{"error":"no cert for domain"}`, "no cert for domain"},
		{"plain text body", 500, "internal failure\n", "internal failure"},
		{"empty body", 502, "", "502"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old, oldClient := TailscaledTransport, tsClient
			t.Cleanup(func() { TailscaledTransport, tsClient = old, oldClient })
			tsClientOnce.Do(func() {})
			tsClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return fakeResponse(tt.status, tt.body), nil
			})}

			body, err := get200(context.Background(), "/localapi/v0/status")
			if err == nil {
				t.Fatalf("get200 returned nil error and body %q for HTTP %d", body, tt.status)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q; want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestCertPairUnexpectedStatus(t *testing.T) {
	old := tsClient
	t.Cleanup(func() { tsClient = old })
	tsClientOnce.Do(func() {})
	tsClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return fakeResponse(500, "cert unavailable"), nil
	})}

	_, _, err := CertPair(context.Background(), "host.example.ts.net")
	if err == nil {
		t.Fatal("CertPair returned nil error")
	}
	if strings.Contains(err.Error(), "no delimiter") {
		t.Errorf("error = %q; want the daemon's failure, not a delimiter error", err)
	}
}
