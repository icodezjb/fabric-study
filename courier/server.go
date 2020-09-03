package courier

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/icodezjb/fabric-study/log"
)

type Server struct {
	server *http.Server
}

type Request struct {
	CrossID  string
	Receipt  string
	Sequence int64
}

func NewServer(port string, h *Handler) *Server {
	s := &Server{}

	s.server = &http.Server{
		Addr:    ":" + port,
		Handler: h,
	}

	return s
}

func (s *Server) Start() {
	go s.serve()
}

func (s *Server) serve() {
	err := s.server.ListenAndServe()
	if err != nil {
		log.Crit("server run failed, err: %v", err)
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	code, msg := http.StatusOK, ""

	switch req.URL.Path {
	case "/v1/receipt":
		if req.Method != "POST" {
			code, msg = http.StatusBadRequest, "support POST request only"
			break
		}

		crossID := req.PostFormValue("crossID")
		receipt := req.PostFormValue("receipt")
		sequence := req.PostFormValue("sequence")

		//TODO check crossID, receipt, sequence
		seq, _ := strconv.Atoi(sequence)

		h.RecvMsg(Request{crossID, receipt, int64(seq)})
	default:
		code = http.StatusNotFound
		msg = fmt.Sprintf("%s not found\n", req.URL.Path)
	}

	w.WriteHeader(code)
	if _, err := w.Write([]byte(msg)); err != nil {
		log.Error("ServeHTTP error: %v", err)
	}
}
