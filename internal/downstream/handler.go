package downstream

import "context"

// provides abstraction of the downstream server for the application
type Server interface {
	StartServer() error
	ShutDown(ctx context.Context) error
}

type Handler struct {
	Server
}

func NewHandler(s Server) *Handler {
	return &Handler{
		Server: s,
	}
}
