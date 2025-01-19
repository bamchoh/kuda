package kuda

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"time"

	"go.bug.st/serial"
)

type Kuda struct {
	PortName  string
	Mode      *serial.Mode
	port      serial.Port
	rx        bytes.Buffer
	isWaitAck bool
}

func (kuda *Kuda) Open() (err error) {
	kuda.port, err = serial.Open(kuda.PortName, kuda.Mode)
	if err != nil {
		return fmt.Errorf("opening serial port was failed: %w", err)
	}
	return nil
}

func (kuda *Kuda) Close() error {
	return kuda.port.Close()
}

func (kuda *Kuda) waitACK() error {
	kuda.isWaitAck = true
	defer func() {
		kuda.isWaitAck = false
	}()

	buf := make([]byte, 1)
	if _, err := kuda.Read(buf); err != nil {
		return err
	}

	return nil
}

func (kuda *Kuda) sendACK() error {
	buf := &bytes.Buffer{}
	if err := binary.Write(buf, binary.LittleEndian, uint32(1)); err != nil {
		return err
	}

	// next: 0
	// status: 0 (ACK)
	if _, err := buf.Write([]byte{0, 0}); err != nil {
		return err
	}

	if _, err := kuda.port.Write(buf.Bytes()); err != nil {
		return err
	}

	return nil
}

func (kuda *Kuda) Write(data []byte) (n int, err error) {
	size := 1024
	var j int
	for i := 0; i < len(data); i += size {
		j += size
		if j > len(data) {
			j = len(data)
		}

		if err := binary.Write(kuda.port, binary.LittleEndian, uint32(j-i)); err != nil {
			return 0, err
		}

		var next byte
		if j < len(data) {
			next = 1
		}

		sendBytes := append([]byte{next}, data[i:j]...)
		if _, err := kuda.port.Write(sendBytes); err != nil {
			return 0, err
		}

		if err := kuda.waitACK(); err != nil {
			return 0, err
		}
	}

	return len(data), nil
}

func (kuda *Kuda) WriteTo(w io.Writer) (n int64, err error) {
	return kuda.rx.WriteTo(w)
}

func (kuda *Kuda) Read(resultBuf []byte) (n int, err error) {
	if kuda.rx.Len() > 0 {
		n := copy(resultBuf, kuda.rx.Next(len(resultBuf)))
		return n, nil
	}

	tmpRxBuf := &bytes.Buffer{}
	buf := make([]byte, 65536)
	var size int32
	var next byte
	for {
		if tmpRxBuf.Len() > 0 {
			kuda.port.SetReadTimeout(1 * time.Second)
		} else {
			kuda.port.SetReadTimeout(serial.NoTimeout)
		}
		n, err := kuda.port.Read(buf)
		if err != nil {
			log.Println("reading buffer was failed:", err)
			kuda.port.Close()
			kuda.port, err = serial.Open(kuda.PortName, kuda.Mode)
			if err != nil {
				return 0, fmt.Errorf("re-opening serial port was failed: %w", err)
			}
			continue
		}

		if tmpRxBuf.Len() > 0 && n == 0 {
			tmpRxBuf.Reset()
			continue
		}

		tmpRxBuf.Write(buf[:n])

		if tmpRxBuf.Len() < 5 {
			continue
		}

		if size == 0 {
			size = int32(binary.LittleEndian.Uint32(tmpRxBuf.Next(4)))
			if next, err = tmpRxBuf.ReadByte(); err != nil {
				return 0, fmt.Errorf("parsing received data was failed: %w", err)
			}
		}

		if tmpRxBuf.Len() < int(size) {
			continue
		}

		if !kuda.isWaitAck {
			if err := kuda.sendACK(); err != nil {
				return 0, err
			}
		}

		if _, err := tmpRxBuf.WriteTo(&kuda.rx); err != nil {
			return 0, fmt.Errorf("draining temporary buffer to kuda.rx was failed: %w", err)
		}

		if next > 0 {
			tmpRxBuf.Reset()
			size = 0
			next = 0
			continue
		}

		n = copy(resultBuf, kuda.rx.Next(len(resultBuf)))

		return n, nil
	}
}
