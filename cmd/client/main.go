package main

import (
	"log"

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

func main() {
	client := kuda.Client{
		PortName: "COM9",
	}

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
