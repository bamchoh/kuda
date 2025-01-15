package kuda

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"go.bug.st/serial"
)

type response struct {
	writer io.Writer
}

func (r *response) Header() http.Header {
	header := make(map[string][]string, 0)
	return header
}

func (r *response) Write(data []byte) (int, error) {
	if err := binary.Write(r.writer, binary.LittleEndian, uint32(len(data))); err != nil {
		return 0, err
	}

	size := 1024
	var j int
	for i := 0; i < len(data); i += size {
		j += size
		if j > len(data) {
			j = len(data)
		}
		if err := binary.Write(r.writer, binary.LittleEndian, data[i:j]); err != nil {
			return 0, err
		}
		time.Sleep(1 * time.Millisecond)
	}

	return len(data), nil
}

func (r *response) WriteHeader(statusCode int) {
}

type Server struct {
	PortName string
	BaudRate int
}

func (s *Server) Serve(ctx context.Context, handler http.Handler) error {
	mode := &serial.Mode{
		BaudRate: s.BaudRate,
	}

	port, err := serial.Open(s.PortName, mode)
	if err != nil {
		return fmt.Errorf("[server] opening serial port was failed: %w", err)
	}
	defer port.Close()

	totalBuf := make([]byte, 0)
	buf := make([]byte, 65536)
	for {
		if len(totalBuf) > 0 {
			port.SetReadTimeout(1 * time.Second)
		} else {
			port.SetReadTimeout(serial.NoTimeout)
		}
		n, err := port.Read(buf)
		if err != nil {
			log.Println("[server] reading buffer was failed:", err)
			port.Close()
			port, err = serial.Open(s.PortName, mode)
			if err != nil {
				return fmt.Errorf("[server] re-opening serial port was failed: %w", err)
			}
			continue
		}

		if len(totalBuf) > 0 && n == 0 {
			totalBuf = make([]byte, 0)
			continue
		}

		totalBuf = append(totalBuf, buf[:n]...)

		if len(totalBuf) < 4 {
			continue
		}

		size := binary.LittleEndian.Uint32(totalBuf[:4])

		if len(totalBuf) < int(size)+4 {
			continue
		}

		buf := &bytes.Buffer{}
		buf.Write(totalBuf[4:])
		req, err := http.NewRequest("POST", "", buf)
		if err != nil {
			log.Println("[server] creating a request was failed:", err)
			continue
		}

		w := &response{
			port,
		}
		handler.ServeHTTP(w, req)

		totalBuf = make([]byte, 0)
	}
}
