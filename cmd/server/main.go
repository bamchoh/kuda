package main

import (
	"flag"
	"log"
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

	FileTransferUploadArgs struct {
		Name string
		Data []byte
	}
)

func (f *FileTransfer) Download(r *http.Request, args *FileTransferArgs, result *FileTransferReply) error {
	data, err := os.ReadFile(args.Name)
	if err != nil {
		return err
	}

	result.Name = args.Name
	result.Data = data

	return nil
}

func (f *FileTransfer) Upload(r *http.Request, args *FileTransferUploadArgs, result *FileTransferReply) error {
	if err := os.WriteFile(args.Name, args.Data, 0777); err != nil {
		return err
	}

	result.Name = args.Name

	return nil
}

func main() {
	portname := flag.String("port", "COM1", "port name")
	flag.Parse()

	s := rpc.NewServer()
	s.RegisterCodec(json2.NewCodec(), "application/json")
	calculator := &Calculator{}
	s.RegisterService(calculator, "")
	filetransfer := &FileTransfer{}
	s.RegisterService(filetransfer, "")

	if err := kuda.Serve(*portname, s); err != nil {
		log.Println(err)
	}
}
