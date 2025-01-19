package main

import (
	"context"
	"net/http"
	"os"

	"github.com/gorilla/rpc/v2"
	"github.com/gorilla/rpc/v2/json2"

	"github.com/bamchoh/kuda"
)

type (
	Calculator   struct{}
	AdditionArgs struct {
		Add, Added int
	}
	AdditionResult struct {
		Computation int
	}
)

func (c Calculator) Add(r *http.Request, args *AdditionArgs, result *AdditionResult) error {
	result.Computation = args.Add + args.Added
	return nil
}

type (
	FileTransfer     struct{}
	FileTransferArgs struct {
		Name string
	}
	FileTransferReply struct {
		Name string
		Data []byte
	}
)

func (f *FileTransfer) Trans(r *http.Request, args *FileTransferArgs, result *FileTransferReply) error {
	data, err := os.ReadFile(args.Name)
	if err != nil {
		return err
	}

	result.Name = args.Name
	result.Data = data

	return nil
}

func main() {
	s := rpc.NewServer()
	s.RegisterCodec(json2.NewCodec(), "")
	calculator := &Calculator{}
	s.RegisterService(calculator, "")
	filetransfer := &FileTransfer{}
	s.RegisterService(filetransfer, "")

	srv := &kuda.Server{
		PortName: "/dev/ttyGS0",
	}
	srv.Serve(context.Background(), s)
}
