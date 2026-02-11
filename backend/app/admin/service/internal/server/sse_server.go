package server

import (
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"github.com/tx7do/kratos-bootstrap/transport/sse"

	sseServer "github.com/tx7do/kratos-transport/transport/sse"
)

// NewSseServer creates a new SSE server.
func NewSseServer(ctx *bootstrap.Context) *sseServer.Server {
	cfg := ctx.GetConfig()

	if cfg == nil || cfg.Server == nil || cfg.Server.Sse == nil {
		return nil
	}

	l := ctx.NewLoggerHelper("sse-server/admin-service")

	srv := sse.NewSseServer(cfg.Server.Sse,
		sseServer.WithSubscriberFunction(func(streamID sseServer.StreamID, sub *sseServer.Subscriber) {
			//l.Infof("SSE: [%s]", sub.URL)
			l.Infof("subscriber [%s] connected", streamID)
		}),
	)

	//srv.CreateStream("test")

	return srv
}
