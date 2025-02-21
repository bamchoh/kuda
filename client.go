package kuda

import (
	"bytes"
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
	port := &Kuda{
		PortName: c.PortName,
		Mode: &serial.Mode{
			BaudRate: c.BaudRate,
		},
	}

	err := port.Open()
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

	fmt.Println("[client] Write")
	if _, err := port.Write(outbuf.Bytes()); err != nil {
		return nil, fmt.Errorf("[client] write error: %w", err)
	}

	fmt.Println("[client] Read")
	rxBuf := &bytes.Buffer{}
	rxBytes := make([]byte, 65535)
	n, err := port.Read(rxBytes)
	if err != nil {
		return nil, fmt.Errorf("[client] reading buffer was failed: %w", err)
	}

	if _, err := rxBuf.Write(rxBytes[:n]); err != nil {
		return nil, fmt.Errorf("[client] appending RX buffer was failed: %w", err)
	}

	if _, err := port.WriteTo(rxBuf); err != nil {
		return nil, fmt.Errorf("[client] draining RX buffer was failed: %w", err)
	}

	var resp JsonRpcResponse
	dec := json.NewDecoder(rxBuf)
	if err := dec.Decode(&resp); err != nil {
		return nil, fmt.Errorf("[client] decode error: %w", err)
	}

	if resp.Error.Code != 0 {
		return nil, fmt.Errorf("[client] error response has been received: %d : %s", resp.Error.Code, resp.Error.Message)
	}

	return &resp, nil
}
