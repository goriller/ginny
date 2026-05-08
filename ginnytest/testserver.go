// Package ginnytest provides test helpers for the Ginny framework.
package ginnytest

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goriller/ginny/v2/server"
)

// TestServer is a test server that wraps a Ginny server for integration testing.
// It uses httptest.Server for the app port and optionally starts the admin port.
type TestServer struct {
	t       *testing.T
	srv     *server.Server
	appTS   *httptest.Server

	client  *http.Client
	baseURL string
}

// NewTestServer creates a new TestServer with the given server options.
// It starts the app and admin servers on random ports via httptest.
func NewTestServer(t *testing.T, opts ...server.Option) (*TestServer, error) {
	t.Helper()

	// Ensure apps listen on random ports
	opts = append(opts,
		server.WithAppAddr(":0"),
		server.WithAdminAddr(":0"),
	)

	srv := server.New(opts...)

	// Start the app server using httptest
	appTS := httptest.NewUnstartedServer(srv.AppMux())
	appTS.EnableHTTP2 = true
	appTS.Start()

	ts := &TestServer{
		t:       t,
		srv:     srv,
		appTS:   appTS,
		client:  appTS.Client(),
		baseURL: appTS.URL,
	}

	// Start admin server in background using the real server's admin mux
	adminMux := srv.AdminMux()
	adminListener, err := newLocalListener()
	if err != nil {
		appTS.Close()
		return nil, err
	}
	go func() {
		_ = http.Serve(adminListener, adminMux)
	}()

	// Mark as ready
	srv.SetReady(true)

	t.Cleanup(func() {
		ts.Close()
	})

	return ts, nil
}

// Client returns an *http.Client configured to talk to the test server.
func (s *TestServer) Client() *http.Client {
	return s.client
}

// BaseURL returns the base URL of the test server (e.g., "http://127.0.0.1:54321").
func (s *TestServer) BaseURL() string {
	return s.baseURL
}

// Close shuts down the test server and releases resources.
func (s *TestServer) Close() {
	if s.appTS != nil {
		s.appTS.Close()
	}
	_ = s.srv.Stop(context.Background())
}

// Server returns the underlying Ginny server for advanced access.
func (s *TestServer) Server() *server.Server {
	return s.srv
}

// newLocalListener creates a TCP listener on a random port.
func newLocalListener() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}
