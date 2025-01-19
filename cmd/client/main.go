package main

import (
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
)

func FileTransferTrans(client kuda.Client) {
	response, err := client.Call("FileTransfer.Trans", &FileTransferArgs{Name: "IMG_9134.mp4"})
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

func main() {
	client := kuda.Client{
		PortName: "COM10",
	}

	FileTransferTrans(client)

}
