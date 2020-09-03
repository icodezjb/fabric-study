package courier

import (
	"fmt"
	"net/http"

	"github.com/icodezjb/fabric-study/log"
)

type Server struct {
	server *http.Server
}

func NewServer(endPoint string, h *Handler) *Server {
	s := &Server{}

	s.server = &http.Server{
		Addr:    endPoint,
		Handler: h,
	}

	log.Info("[Server] http server to listen", "endPoint", "http://"+endPoint)
	return s
}

func (s *Server) Start() {
	go s.serve()
	log.Info("[Server] http server started")
}

func (s *Server) serve() {
	err := s.server.ListenAndServe()
	if err != nil {
		log.Info(fmt.Sprintf("[Server] %s", err))
	}
	log.Info("[Server] http server stopped")
}

func (s *Server) Stop() {
	log.Info("[Server] http server stopping")
	if err := s.server.Shutdown(nil); err != nil {
		log.Error("[Server] shutdown", "err", err)
	}
}
