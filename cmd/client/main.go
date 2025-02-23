package main

import (
	"flag"
	"log"
	"os"

	"github.com/bamchoh/kuda"
)

type (
	AdditionArgs struct {
		Add, Added int
	}
	AdditionResult struct {
		Computation int
	}
)

func CalculatorAdd(client kuda.Client) {
	added := 10
	add := 12
	response, err := client.Call("Calculator.Add", &AdditionArgs{Added: added, Add: add})
	if err != nil {
		log.Fatalln(err)
	}
	var result AdditionResult
	err = response.GetObject(&result) // (2)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("%d + %d = %d", added, add, result.Computation)
}

type (
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

func FileTransferDownload(client kuda.Client) {
	response, err := client.Call("FileTransfer.Download", &FileTransferArgs{Name: "main.go"})
	if err != nil {
		log.Fatalln(err)
	}
	var result FileTransferReply
	err = response.GetObject(&result)
	if err != nil {
		log.Fatalln(err)
	}

	err = os.WriteFile(result.Name, result.Data, 0777)
	if err != nil {
		log.Fatalln(err)
	}
}

func FileTransferUpload(client kuda.Client) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		log.Fatalln(err)
	}

	response, err := client.Call("FileTransfer.Upload", &FileTransferUploadArgs{Name: "main.go", Data: data})
	if err != nil {
		log.Fatalln(err)
	}
	var result FileTransferReply
	err = response.GetObject(&result)
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	portname := flag.String("port", "COM1", "port name")
	flag.Parse()

	client := kuda.Client{
		PortName: *portname,
	}

	// CalculatorAdd(client)
	FileTransferUpload(client)

	FileTransferDownload(client)
}
