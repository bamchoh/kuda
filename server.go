package kuda

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"go.bug.st/serial"
)

func Serve(portname string, handler http.Handler) error {
	port := &Kuda{
		PortName: portname,
		Mode: &serial.Mode{
			BaudRate: 115200,
		},
	}

	server := &Server{port}
	return server.Serve(handler)
}

type response struct {
	writer io.Writer
	err    error
}

func (r *response) Header() http.Header {
	header := make(map[string][]string, 0)
	return header
}

func (r *response) Write(data []byte) (int, error) {
	n, err := r.writer.Write(data)
	if err != nil {
		r.err = err
	}
	return n, nil
}

func (r *response) WriteHeader(statusCode int) {
}

func (r *response) Err() error {
	return r.err
}

type Server struct {
	port *Kuda
}

func (s *Server) Serve(handler http.Handler) error {
	if err := s.port.Open(); err != nil {
		return fmt.Errorf("[server] opening serial port was failed: %w", err)
	}
	defer s.port.Close()

	for {
		packet, err := s.port.ReadPacket()
		if err != nil {
			return fmt.Errorf("[server] reading request was failed: %w", err)
		}

		req, err := http.NewRequest("POST", "", packet)
		if err != nil {
			return fmt.Errorf("[server] creating a request was failed: %w", err)
		}

		w := &response{
			s.port,
			nil,
		}

		handler.ServeHTTP(w, req)

		if w.Err() != nil {
			log.Println("[server] ServeHTTP error:", w.Err())
		}
	}
}
