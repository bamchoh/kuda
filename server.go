package kuda

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"go.bug.st/serial"
)

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
	PortName string
	BaudRate int
}

func (s *Server) Serve(ctx context.Context, handler http.Handler) error {
	port := &Kuda{
		PortName: s.PortName,
		Mode: &serial.Mode{
			BaudRate: s.BaudRate,
		},
	}

	if err := port.Open(); err != nil {
		return fmt.Errorf("[server] opening serial port was failed: %w", err)
	}
	defer port.Close()

	rxBuf := &bytes.Buffer{}
	for {
		rxBytes := make([]byte, 65535)
		n, err := port.Read(rxBytes)
		fmt.Println("[server] Read", n, "bytes")
		if err != nil {
			return fmt.Errorf("[server] reading buffer was failed: %w", err)
		}

		if _, err := rxBuf.Write(rxBytes[:n]); err != nil {
			return fmt.Errorf("[server] appending RX buffer was failed: %w", err)
		}

		if _, err := port.WriteTo(rxBuf); err != nil {
			return fmt.Errorf("[server] draining RX buffer was failed: %w", err)
		}

		req, err := http.NewRequest("POST", "", rxBuf)
		if err != nil {
			return fmt.Errorf("[server] creating a request was failed: %w", err)
		}

		w := &response{
			port,
			nil,
		}
		handler.ServeHTTP(w, req)

		if w.Err() != nil {
			log.Println("[server] ServeHTTP error:", w.Err())
		}

		rxBuf.Reset()
	}
}
