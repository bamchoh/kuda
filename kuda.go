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

type Kuda struct {
	PortName  string
	Mode      *serial.Mode
	WriteSize int

	port      serial.Port
	rx        *bytes.Buffer
	isWaitAck bool
	rxTimeout time.Duration
}

func (kuda *Kuda) Open() (err error) {
	kuda.port, err = serial.Open(kuda.PortName, kuda.Mode)
	if err != nil {
		return fmt.Errorf("opening serial port was failed: %w", err)
	}
	kuda.rx = &bytes.Buffer{}
	kuda.isWaitAck = false
	kuda.rxTimeout = serial.NoTimeout
	if kuda.WriteSize == 0 {
		kuda.WriteSize = 1024
	}
	return nil
}

func (kuda *Kuda) Close() error {
	return kuda.port.Close()
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
	kuda.isWaitAck = true
	origTimeout := kuda.rxTimeout
	kuda.rxTimeout = 1 * time.Second
	defer func() {
		kuda.isWaitAck = false
		kuda.rxTimeout = origTimeout
	}()

	buf := make([]byte, 1)
	fmt.Println("[waitACK] start")
	defer fmt.Println("[waitACK] end")
	if _, err := kuda.Read(buf); err != nil {
		return err
	}

	return nil
}

func (kuda *Kuda) sendACK() error {
	buf := &bytes.Buffer{}
	if err := binary.Write(buf, binary.BigEndian, uint32(1)); err != nil {
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
	size := kuda.WriteSize
	j := 0
	for i := 0; i < len(data); i += size {
		j += size
		if j > len(data) {
			j = len(data)
		}

		if err := binary.Write(kuda.port, binary.BigEndian, uint32(j-i)); err != nil {
			return 0, err
		}

		var next byte
		if j < len(data) {
			next = 1
		}

		sendBytes := append([]byte{next}, data[i:j]...)
		fmt.Println("Write", len(sendBytes), "bytes")
		// dumpByteSlice(sendBytes)
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

func (kuda *Kuda) internalRead(tmpRxBuf *bytes.Buffer, buf []byte) (int, error) {
	if tmpRxBuf.Len() > 0 {
		kuda.port.SetReadTimeout(1 * time.Second)
	} else {
		kuda.port.SetReadTimeout(kuda.rxTimeout)
	}
	fmt.Println("[kuda.Timeout] kuda.rxTimeout", kuda.rxTimeout)
	n, err := kuda.port.Read(buf)
	if err != nil {
		return n, err
	}

	if n == 0 {
		return n, errors.New("timeout error was happened")
	}
	return n, nil
}

func (kuda *Kuda) Read(resultBuf []byte) (n int, err error) {
	fmt.Println("[Kuda.Read] kuda.rx.Len", kuda.rx.Len())

	tmpRxBuf := &bytes.Buffer{}
	buf := make([]byte, 65536)
	var size int32
	var next byte
	first := true
	for {
		err = nil

		if first && kuda.rx.Len() > 0 {
			fmt.Println("[Kuda.Read] copy kuda.rx")
			n = copy(buf, kuda.rx.Next(len(buf)))
			err = nil
		} else {
			fmt.Println("[Kuda.Read] read data from internal reader")
			n, err = kuda.internalRead(tmpRxBuf, buf)
		}
		first = false
		if err != nil {
			kuda.Reopen()
			return 0, fmt.Errorf("reading buffer was failed:%w", err)
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
			size = int32(binary.BigEndian.Uint32(tmpRxBuf.Next(4)))
			if next, err = tmpRxBuf.ReadByte(); err != nil {
				kuda.Reopen()
				return 0, fmt.Errorf("parsing received data was failed: %w", err)
			}
		}

		if tmpRxBuf.Len() < int(size) {
			continue
		}

		if !kuda.isWaitAck {
			fmt.Println("[sendACK()]")
			if err := kuda.sendACK(); err != nil {
				kuda.Reopen()
				return 0, err
			}
			fmt.Println("[endACK()]")
		}

		if _, err := tmpRxBuf.WriteTo(kuda.rx); err != nil {
			kuda.Reopen()
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
