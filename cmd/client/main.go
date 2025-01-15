package main

import (
	"log"
	"os"

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

	FileInfoArgs struct {
		Name string
		Data []byte
	}

	FileTransferResult struct {
		Message string
	}
)

func main() {
	client := kuda.Client{
		PortName: "COM9",
	}

	/*
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
	*/

	filename := "IMG_9134.mp4"
	mainGoData, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalln(err)
	}

	response, err := client.Call("FileTransfer.Trans", &FileInfoArgs{
		Name: filename,
		Data: mainGoData,
	})
	if err != nil {
		log.Fatalln(err)
	}

	var result FileTransferResult
	if err = response.GetObject(&result); err != nil {
		log.Fatalln(err)
	}

	if result.Message == "" {
		log.Println("result: OK")
	} else {
		log.Println("error: ", result.Message)
	}
}
