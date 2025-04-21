package agent

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1/v1poc1connect"
)

// NewInMemoryAgentClient spins up an in‑memory HTTP/2 server+client pair
// for testing streaming RPCs against the given AgentServiceHandler.
func NewInMemoryAgentClient(
	ctx context.Context,
	srvImpl v1poc1connect.AgentServiceHandler,
) (
	client v1poc1connect.AgentServiceClient,
	cleanup func(),
) {
	// 1) Build the handler and mount under its generated path
	path, handler := v1poc1connect.NewAgentServiceHandler(srvImpl)
	mux := http.NewServeMux()
	mux.Handle(path, handler)

	// 2) Create an unstarted test server so we can enable HTTP/2
	ts := httptest.NewUnstartedServer(mux)
	ts.EnableHTTP2 = true
	ts.StartTLS() // now serves HTTP/2 over TLS with a test cert

	// 3) Wire up the generated client to talk to our in‑memory server
	client = v1poc1connect.NewAgentServiceClient(ts.Client(), ts.URL)

	// 4) Provide cleanup to shut down the server
	cleanup = func() {
		ts.Close()
	}
	return client, cleanup
}
