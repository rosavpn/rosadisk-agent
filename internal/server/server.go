package server

import (
	_ "embed"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"rosadisk-agent/api"
	"rosadisk-agent/api/gen"
)

//go:embed docs.html
var docsHTML []byte

type Server struct {
	Echo *echo.Echo
}

func NewServer() *Server {
	e := echo.New()
	s := &Server{Echo: e}

	gen.RegisterHandlers(e, s)
	e.GET("/docs", s.GetDocs)

	return s
}

func (s *Server) Start(addr string) error {
	return s.Echo.Start(addr)
}

func (s *Server) GetHealth(ctx echo.Context) error {
	resp := gen.HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
	}
	return ctx.JSON(http.StatusOK, resp)
}

func (s *Server) GetOpenAPIJSON(ctx echo.Context) error {
	specJSON, err := gen.GetSpecJSON()
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return ctx.Blob(http.StatusOK, "application/json", specJSON)
}

func (s *Server) GetOpenAPIYAML(ctx echo.Context) error {
	return ctx.Blob(http.StatusOK, "text/yaml", api.OpenAPIYAML)
}

func (s *Server) GetDocs(ctx echo.Context) error {
	return ctx.HTML(http.StatusOK, string(docsHTML))
}
