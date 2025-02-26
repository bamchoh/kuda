package kuda

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/bamchoh/kuda/internal/testutil"
	"go.bug.st/serial"
)

type DummyPort struct {
	InnerRxBuffer io.ReadWriter
	InnerTxBuffer io.ReadWriter
	rxTimeout     time.Duration
	closed        bool
}

func (dp *DummyPort) SetMode(mode *serial.Mode) error { return nil }
func (dp *DummyPort) Read(p []byte) (n int, err error) {
	timeout := time.Now().Add(dp.rxTimeout)
	for dp.rxTimeout == serial.NoTimeout || time.Now().Before(timeout) {
		if n, err := dp.InnerRxBuffer.Read(p); err != nil {
			if dp.closed {
				return 0, errors.New("port was closed")
			}
			if err == io.EOF {
				continue
			}
			return n, err
		} else {
			return n, err
		}
	}
	return 0, nil
}
func (dp *DummyPort) Write(p []byte) (n int, err error) {
	return dp.InnerTxBuffer.Write(p)
}
func (dp *DummyPort) Drain() error             { return nil }
func (dp *DummyPort) ResetInputBuffer() error  { return nil }
func (dp *DummyPort) ResetOutputBuffer() error { return nil }
func (dp *DummyPort) SetDTR(dtr bool) error    { return nil }
func (dp *DummyPort) SetRTS(rts bool) error    { return nil }
func (dp *DummyPort) GetModemStatusBits() (*serial.ModemStatusBits, error) {
	return &serial.ModemStatusBits{}, nil
}
func (dp *DummyPort) SetReadTimeout(t time.Duration) error {
	dp.rxTimeout = t
	return nil
}
func (dp *DummyPort) Close() error {
	dp.closed = true
	return nil
}
func (dp *DummyPort) Break(time.Duration) error { return nil }

func newOpenSerialFunc(rxbuf io.ReadWriter, txbuf io.ReadWriter) func() {
	t := openSerial
	openSerial = func(portname string, mode *serial.Mode) (serial.Port, error) {
		return &DummyPort{InnerRxBuffer: rxbuf, InnerTxBuffer: txbuf}, nil
	}
	return func() {
		openSerial = t
	}
}

func makePacket(buf io.ReadWriter, next int, body []byte) error {
	if err := binary.Write(buf, binary.BigEndian, uint32(len(body))); err != nil {
		return err
	}

	// next
	if err := binary.Write(buf, binary.BigEndian, uint8(next)); err != nil {
		return err
	}

	// body
	if err := binary.Write(buf, binary.BigEndian, []byte(body)); err != nil {
		return err
	}

	return nil
}

func TestOpen(t *testing.T) {
	buf := &bytes.Buffer{}
	defer newOpenSerialFunc(buf, buf)()
	kuda := &Kuda{
		PortName: "COM1",
		Mode: &serial.Mode{
			BaudRate: 115200,
		},
	}
	err := kuda.Open()
	defer kuda.Close()
	if err != nil {
		t.Errorf("kuda.Open was failed: %v", err)
	}
}

func TestRead(t *testing.T) {
	rxbuf := &bytes.Buffer{}
	txbuf := &bytes.Buffer{}
	defer newOpenSerialFunc(rxbuf, txbuf)()
	kuda := &Kuda{
		PortName: "COM1",
		Mode: &serial.Mode{
			BaudRate: 115200,
		},
	}
	err := kuda.Open()
	defer kuda.Close()
	if err != nil {
		t.Errorf("kuda.Open was failed: %v", err)
	}

	body := "test"
	if err := makePacket(rxbuf, 0, []byte(body)); err != nil {
		t.Errorf("Making packet was failed: %v", err)
	}

	if packet, err := kuda.ReadPacket(); err != nil {
		t.Errorf("Read was failed: %v", err)
	} else {
		if packet.String() != body {
			t.Errorf("Read content is not match\nwant: %s\ngot:  %s", body, packet.String())
		}

		expectedAckData := []byte{0, 0, 0, 1, 0, 0}
		if !bytes.Equal(txbuf.Bytes(), expectedAckData) {
			t.Errorf("ACK reply is not correct format:\nwant: %v\ngot:  %v", expectedAckData, txbuf.Bytes())
		}
	}
}

func TestWrite(t *testing.T) {
	rxbuf := &testutil.SafeBuffer{}
	txbuf := &testutil.SafeBuffer{}
	defer newOpenSerialFunc(rxbuf, txbuf)()
	kuda := &Kuda{
		PortName: "COM1",
		Mode: &serial.Mode{
			BaudRate: 115200,
		},
	}
	err := kuda.Open()
	defer kuda.Close()
	if err != nil {
		t.Errorf("kuda.Open was failed: %v", err)
	}

	body := "test"

	rxbuf.Write([]byte{0, 0, 0, 1, 0, 0})

	if n, err := kuda.Write([]byte(body)); err != nil {
		t.Errorf("Read was failed: %v", err)
	} else {
		if n != len(body) {
			t.Errorf("Read count is not match (want: %d, got: %d)", len(body), n)
		}

		wantBuffer := &bytes.Buffer{}
		if err := makePacket(wantBuffer, 0, []byte(body)); err != nil {
			t.Errorf("Making packet was failed: %v", err)
		}

		if !bytes.Equal(txbuf.Bytes(), wantBuffer.Bytes()) {
			t.Errorf("Write packet is not correct format:\nwant: %v\ngot:  %v", wantBuffer, txbuf.Bytes())
		}
	}
}

func TestRead_1024bytes(t *testing.T) {
	rxbuf := &testutil.SafeBuffer{}
	txbuf := &testutil.SafeBuffer{}
	defer newOpenSerialFunc(rxbuf, txbuf)()

	kuda := &Kuda{
		PortName: "COM1",
		Mode: &serial.Mode{
			BaudRate: 115200,
		},
	}
	err := kuda.Open()
	defer kuda.Close()
	if err != nil {
		t.Errorf("kuda.Open was failed: %v", err)
	}

	body, err := testutil.MakeRandomStr(1024)
	if err != nil {
		t.Errorf("Making test body was failed: %v", err)
	}

	if err := makePacket(rxbuf, 0, []byte(body)); err != nil {
		t.Errorf("Making packet was failed: %v", err)
	}

	if packet, err := kuda.ReadPacket(); err != nil {
		t.Errorf("Read was failed: %v", err)
	} else {
		if packet.String() != body {
			t.Errorf("Read content is not match\nwant: %s\ngot:  %s", body, packet.String())
		}

		expectedAckData := []byte{0, 0, 0, 1, 0, 0}
		if !bytes.Equal(txbuf.Bytes(), expectedAckData) {
			t.Errorf("ACK reply is not correct format:\nwant: %v\ngot:  %v", expectedAckData, txbuf.Bytes())
		}
	}
}

func TestRead_1025bytes(t *testing.T) {
	rxbuf := &testutil.SafeBuffer{}
	txbuf := &testutil.SafeBuffer{}
	defer newOpenSerialFunc(rxbuf, txbuf)()
	kuda := &Kuda{
		PortName: "COM1",
		Mode: &serial.Mode{
			BaudRate: 115200,
		},
	}
	err := kuda.Open()
	if err != nil {
		t.Errorf("kuda.Open was failed: %v", err)
	}

	body, err := testutil.MakeRandomStr(1025)
	if err != nil {
		t.Errorf("Making test body was failed: %v", err)
	}

	if err := makePacket(rxbuf, 0, []byte(body)); err != nil {
		t.Errorf("Making packet was failed: %v", err)
	}

	if packet, err := kuda.ReadPacket(); err != nil {
		t.Errorf("Read was failed: %v", err)
	} else {
		if packet.String() != body {
			t.Errorf("Read content is not match\nwant: %s\ngot:  %s", body, packet.String())
		}

		expectedAckData := []byte{0, 0, 0, 1, 0, 0}
		if !bytes.Equal(txbuf.Bytes(), expectedAckData) {
			t.Errorf("ACK reply is not correct format:\nwant: %v\ngot:  %v", expectedAckData, txbuf.Bytes())
		}
	}
}
