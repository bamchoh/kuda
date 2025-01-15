package kuda

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"go.bug.st/serial"
)

type JsonRpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type JsonRpcRequest struct {
	Method  string `json:"method"`
	Params  any    `json:"params"`
	Id      int    `json:"id"`
	Version string `json:"jsonrpc"`
}

type JsonRpcResponse struct {
	Result  *json.RawMessage `json:"result"`
	Id      int              `json:"id"`
	Version string           `json:"jsonrpc"`
	Error   JsonRpcError     `json:"error"`
}

func (response *JsonRpcResponse) GetObject(data any) error {
	return json.Unmarshal(*response.Result, data)
}

type Client struct {
	PortName string
	BaudRate int
}

func (c *Client) Call(method string, params any) (*JsonRpcResponse, error) {
	mode := &serial.Mode{
		BaudRate: c.BaudRate,
	}

	port, err := serial.Open(c.PortName, mode)
	if err != nil {
		return nil, fmt.Errorf("[client] serial port couldn't be opened: %w", err)
	}
	defer port.Close()

	packet := &JsonRpcRequest{
		Method:  method,
		Params:  params,
		Id:      0,
		Version: "2.0",
	}

	outbuf := &bytes.Buffer{}
	enc := json.NewEncoder(outbuf)
	if err := enc.Encode(packet); err != nil {
		return nil, fmt.Errorf("[client] encode error: %w", err)
	}

	b := outbuf.Bytes()
	binary.Write(port, binary.LittleEndian, uint32(len(b)))
	binary.Write(port, binary.LittleEndian, b)

	totalBuf := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := port.Read(buf)
		if err != nil {
			return nil, fmt.Errorf("[client] reading buffer was failed: %w", err)
		}
		totalBuf = append(totalBuf, buf[:n]...)

		if len(totalBuf) < 4 {
			continue
		}

		size := binary.LittleEndian.Uint32(totalBuf[:4])

		if len(totalBuf) < int(size) {
			continue
		}

		buf := &bytes.Buffer{}
		buf.Write(totalBuf[4:])

		var resp JsonRpcResponse
		dec := json.NewDecoder(buf)
		if err := dec.Decode(&resp); err != nil {
			return nil, fmt.Errorf("[client] decode error: %w", err)
		}

		if resp.Error.Code != 0 {
			return nil, fmt.Errorf("[client] error response has been received: %d : %s", resp.Error.Code, resp.Error.Message)
		}

		return &resp, nil
	}
}
