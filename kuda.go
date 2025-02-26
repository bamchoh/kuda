package kuda

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"go.bug.st/serial"
)

type Packet struct {
	Data []byte
	Next byte
}

type Kuda struct {
	PortName  string
	Mode      *serial.Mode
	WriteSize int

	rxPacketQueue chan *Packet
	rxBuffer      *bytes.Buffer
	cancelRead    context.CancelFunc
	readCanceled  chan struct{}
	port          serial.Port
	isWaitAck     bool
	rxTimeout     time.Duration
}

var openSerial = func(portname string, mode *serial.Mode) (serial.Port, error) {
	return serial.Open(portname, mode)
}

func (kuda *Kuda) Open() (err error) {
	kuda.port, err = openSerial(kuda.PortName, kuda.Mode)
	if err != nil {
		return fmt.Errorf("opening serial port was failed: %w", err)
	}
	kuda.rxPacketQueue = make(chan *Packet, 10)
	kuda.readCanceled = make(chan struct{})
	kuda.rxBuffer = &bytes.Buffer{}
	kuda.isWaitAck = false
	kuda.rxTimeout = serial.NoTimeout
	if kuda.WriteSize == 0 {
		kuda.WriteSize = 1024
	}

	var ctx context.Context
	ctx, kuda.cancelRead = context.WithCancel(context.Background())

	go func(ctx context.Context) {
		for {
			if err := kuda.read(ctx); err != nil {
				select {
				case <-ctx.Done():
					kuda.readCanceled <- struct{}{}
					return
				default:
				}

				kuda.Reopen()
			} else {
				kuda.readCanceled <- struct{}{}
				return
			}
		}
	}(ctx)

	return nil
}

func (kuda *Kuda) Close() error {
	if kuda.cancelRead != nil {
		kuda.cancelRead()
	}

	err := kuda.port.Close()

	select {
	case <-kuda.readCanceled:
	case <-time.After(3 * time.Second):
	}

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
	kuda.isWaitAck = true
	origTimeout := kuda.rxTimeout
	kuda.rxTimeout = 1 * time.Second
	defer func() {
		kuda.isWaitAck = false
		kuda.rxTimeout = origTimeout
	}()

	<-kuda.rxPacketQueue

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
		packet := <-kuda.rxPacketQueue
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

func (kuda *Kuda) read(ctx context.Context) (err error) {
	readBytes := make([]byte, 2048)
	var size int32
	var next byte
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		var n int

		n, err = kuda.internalRead(kuda.rxBuffer.Len(), readBytes)

		if err != nil {
			return fmt.Errorf("reading buffer was failed:%w", err)
		}

		if kuda.rxBuffer.Len() > 0 {
			kuda.rxBuffer.Reset()
			continue
		}

		kuda.rxBuffer.Write(readBytes[:n])

		if kuda.rxBuffer.Len() < 5 {
			continue
		}

		if size == 0 {
			size = int32(binary.BigEndian.Uint32(kuda.rxBuffer.Next(4)))
			if next, err = kuda.rxBuffer.ReadByte(); err != nil {
				return fmt.Errorf("parsing received data was failed: %w", err)
			}
		}

		if kuda.rxBuffer.Len() < int(size) {
			continue
		}

		packet := &Packet{Data: kuda.rxBuffer.Next(int(size)), Next: next}

		kuda.rxPacketQueue <- packet
	}
}
