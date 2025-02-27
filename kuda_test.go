package kuda

import (
	"bytes"
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
	if _, err := sendPacket(rxbuf, 0, []byte(body)); err != nil {
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

	if _, err := sendPacket(rxbuf, 0, []byte(body)); err != nil {
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

	if _, err := sendPacket(rxbuf, 0, []byte(body)); err != nil {
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

func TestRead_65535bytes(t *testing.T) {
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

	body, err := testutil.MakeRandomStr(65535)
	if err != nil {
		t.Errorf("Making test body was failed: %v", err)
	}

	if _, err := sendPacket(rxbuf, 0, []byte(body)); err != nil {
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

func TestRead_slowReceive(t *testing.T) {
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

	body, err := testutil.MakeRandomStr(10)
	if err != nil {
		t.Errorf("Making test body was failed: %v", err)
	}

	go func() {
		tmpBuf := &bytes.Buffer{}
		byteBody := []byte(body)
		if _, err := sendPacket(tmpBuf, 0, byteBody); err != nil {
			t.Errorf("Making packet was failed: %v", err)
		}

		for tmpBuf.Len() > 0 {
			rxbuf.Write(tmpBuf.Next(1))
			time.Sleep(1 * time.Nanosecond)
		}
	}()

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

	body, err := testutil.MakeRandomStr(1024)
	if err != nil {
		t.Errorf("Making test body was failed: %v", err)
	}

	rxbuf.Write([]byte{0, 0, 0, 1, 0, 0})

	if n, err := kuda.Write([]byte(body)); err != nil {
		t.Errorf("Read was failed: %v", err)
	} else {
		if n != len(body) {
			t.Errorf("Read count is not match (want: %d, got: %d)", len(body), n)
		}

		wantBuffer := &bytes.Buffer{}
		if _, err := sendPacket(wantBuffer, 0, []byte(body)); err != nil {
			t.Errorf("Making packet was failed: %v", err)
		}

		if !bytes.Equal(txbuf.Bytes(), wantBuffer.Bytes()) {
			t.Errorf("Write packet is not correct format:\nwant: %v\ngot:  %v", wantBuffer.Bytes(), txbuf.Bytes())
		}
	}
}

func Test_1024Write(t *testing.T) {
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

	rxbuf.Write([]byte{0, 0, 0, 1, 0, 0})

	if n, err := kuda.Write([]byte(body)); err != nil {
		t.Errorf("Read was failed: %v", err)
	} else {
		if n != len(body) {
			t.Errorf("Read count is not match (want: %d, got: %d)", len(body), n)
		}

		wantBuffer := &bytes.Buffer{}
		if _, err := sendPacket(wantBuffer, 0, []byte(body)); err != nil {
			t.Errorf("Making packet was failed: %v", err)
		}

		if !bytes.Equal(txbuf.Bytes(), wantBuffer.Bytes()) {
			t.Errorf("Write packet is not correct format:\nwant: %v\ngot:  %v", wantBuffer.Bytes(), txbuf.Bytes())
		}
	}
}

func Test_1025Write(t *testing.T) {
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

	body, err := testutil.MakeRandomStr(1025)
	if err != nil {
		t.Errorf("Making test body was failed: %v", err)
	}

	rxbuf.Write([]byte{0, 0, 0, 1, 0, 1})
	rxbuf.Write([]byte{0, 0, 0, 1, 0, 2})

	if n, err := kuda.Write([]byte(body)); err != nil {
		t.Errorf("Write was failed: %v", err)
	} else {
		if n != len(body) {
			t.Errorf("Read count is not match (want: %d, got: %d)", len(body), n)
		}

		wantBuffer := &bytes.Buffer{}
		bytesBody := []byte(body)
		_1stBody := bytesBody[:1024]
		if _, err := sendPacket(wantBuffer, 1, _1stBody); err != nil {
			t.Errorf("Making packet was failed: %v", err)
		}
		_2ndBody := bytesBody[1024:1025]
		if _, err := sendPacket(wantBuffer, 0, _2ndBody); err != nil {
			t.Errorf("Making packet was failed: %v", err)
		}

		if !bytes.Equal(txbuf.Bytes(), wantBuffer.Bytes()) {
			t.Errorf("Write packet is not correct format:\nwant: %v\ngot:  %v", wantBuffer.Bytes(), txbuf.Bytes())
		}
	}
}
