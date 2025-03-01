package kuda

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"go.bug.st/serial"
)

type Packet struct {
	Data []byte
	Next byte
}

func sendPacket(buf io.Writer, next byte, body []byte) (int, error) {
	if err := binary.Write(buf, binary.BigEndian, uint32(len(body))); err != nil {
		return 0, err
	}

	// next
	if err := binary.Write(buf, binary.BigEndian, next); err != nil {
		return 0, err
	}

	// body
	if err := binary.Write(buf, binary.BigEndian, []byte(body)); err != nil {
		return 0, err
	}

	return len(body), nil
}

type Kuda struct {
	PortName  string
	Mode      *serial.Mode
	WriteSize int

	rxBuffer  *bytes.Buffer
	port      serial.Port
	rxTimeout time.Duration
}

var openSerial = func(portname string, mode *serial.Mode) (serial.Port, error) {
	return serial.Open(portname, mode)
}

func (kuda *Kuda) Open() (err error) {
	kuda.port, err = openSerial(kuda.PortName, kuda.Mode)
	if err != nil {
		return fmt.Errorf("opening serial port was failed: %w", err)
	}
	kuda.rxBuffer = &bytes.Buffer{}
	kuda.rxTimeout = serial.NoTimeout
	if kuda.WriteSize == 0 {
		kuda.WriteSize = 1024
	}

	return nil
}

func (kuda *Kuda) Close() error {
	err := kuda.port.Close()

	return err
}

func (kuda *Kuda) Reopen() error {
	if err := kuda.Close(); err != nil {
		return fmt.Errorf("reopening was failed:%w", err)
	}

	if err := kuda.Open(); err != nil {
		return fmt.Errorf("reopening was failed:%w", err)
	}

	return nil
}

func (kuda *Kuda) waitACK() error {
	origTimeout := kuda.rxTimeout
	kuda.rxTimeout = 1 * time.Second
	defer func() {
		kuda.rxTimeout = origTimeout
	}()

	_, err := kuda.read()

	return err
}

func (kuda *Kuda) sendACK() error {
	if _, err := sendPacket(kuda.port, 0, []byte{0}); err != nil {
		return err
	}

	return nil
}

func (kuda *Kuda) Write(data []byte) (n int, err error) {
	j := 0
	for i := 0; i < len(data); i = j {
		var next byte
		if i+kuda.WriteSize >= len(data) {
			j = len(data)
		} else {
			j = i + kuda.WriteSize
			next = 1
		}

		if _, err := sendPacket(kuda.port, next, data[i:j]); err != nil {
			return 0, err
		}

		if err := kuda.waitACK(); err != nil {
			return 0, err
		}
	}

	return len(data), nil
}

func (kuda *Kuda) internalRead(tmpRxBufLen int, readBytes []byte) (int, error) {
	if tmpRxBufLen > 0 {
		kuda.port.SetReadTimeout(1 * time.Second)
	} else {
		kuda.port.SetReadTimeout(kuda.rxTimeout)
	}
	n, err := kuda.port.Read(readBytes)
	if err != nil {
		return n, err
	}

	if n == 0 {
		return n, errors.New("timeout error was happened")
	}
	return n, nil
}

func (kuda *Kuda) ReadPacket() (*bytes.Buffer, error) {
	entirePacket := &bytes.Buffer{}
	for {
		packet, err := kuda.read()
		if err != nil {
			return nil, fmt.Errorf("[kuda.ReadPacket] read error: %w", err)
		}
		if _, err := entirePacket.Write(packet.Data); err != nil {
			return nil, fmt.Errorf("[kuda.ReadPacket] writing packet error: %w", err)
		}

		if err := kuda.sendACK(); err != nil {
			return nil, fmt.Errorf("[kuda.ReadPacket] sendACK error: %w", err)
		}

		if packet.Next == 0 {
			return entirePacket, nil
		}
	}
}

func (kuda *Kuda) read() (packet *Packet, err error) {
	readBytes := make([]byte, 2048)
	var size int32
	var next byte
	first := true
	for {
		var n int

		if !(first && kuda.rxBuffer.Len() > 0) {
			n, err = kuda.internalRead(kuda.rxBuffer.Len(), readBytes)
			if err != nil {
				return nil, fmt.Errorf("reading buffer was failed:%w", err)
			}
		}
		first = false

		kuda.rxBuffer.Write(readBytes[:n])

		if size == 0 {
			if kuda.rxBuffer.Len() < 5 {
				continue
			}

			size = int32(binary.BigEndian.Uint32(kuda.rxBuffer.Next(4)))
			if next, err = kuda.rxBuffer.ReadByte(); err != nil {
				return nil, fmt.Errorf("parsing received data was failed: %w", err)
			}
		}

		if kuda.rxBuffer.Len() < int(size) {
			continue
		}

		packet = &Packet{Data: kuda.rxBuffer.Next(int(size)), Next: next}

		return packet, nil
	}
}
