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

	if err := port.Open(); err != nil {
		return nil, fmt.Errorf("[client] serial port couldn't be opened: %w", err)
	}
	defer port.Close()

	rcpReq := &JsonRpcRequest{
		Method:  method,
		Params:  params,
		Id:      0,
		Version: "2.0",
	}

	outbuf := &bytes.Buffer{}
	enc := json.NewEncoder(outbuf)
	if err := enc.Encode(rcpReq); err != nil {
		return nil, fmt.Errorf("[client] encode error: %w", err)
	}

	if _, err := port.Write(outbuf.Bytes()); err != nil {
		return nil, fmt.Errorf("[client] write error: %w", err)
	}

	if packet, err := port.ReadPacket(); err != nil {
		return nil, fmt.Errorf("[client] reading buffer was failed: %w", err)
	} else {
		var resp JsonRpcResponse
		dec := json.NewDecoder(packet)
		if err := dec.Decode(&resp); err != nil {
			return nil, fmt.Errorf("[client] decode error: %w", err)
		}

		if resp.Error.Code != 0 {
			return nil, fmt.Errorf("[client] error response has been received: %d : %s", resp.Error.Code, resp.Error.Message)
		}

		return &resp, nil
	}
}
