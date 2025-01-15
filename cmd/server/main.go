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
	FileTransfer struct{}

	FileInfoArgs struct {
		Name string
		Data []byte
	}

	FileTransferResult struct {
		Message string
	}
)

func (ft FileTransfer) Trans(r *http.Request, args *FileInfoArgs, result *FileTransferResult) error {
	fp, err := os.Create(args.Name)
	if err != nil {
		return err
	}
	defer fp.Close()

	_, err = fp.Write(args.Data)
	return err
}

func main() {
	s := rpc.NewServer()
	s.RegisterCodec(json2.NewCodec(), "")
	calculator := &Calculator{}
	s.RegisterService(calculator, "")
	fileTransfer := &FileTransfer{}
	s.RegisterService(fileTransfer, "")

	srv := &kuda.Server{
		PortName: "/dev/ttyGS0",
	}
	srv.Serve(context.Background(), s)
}
