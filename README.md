# Kuda

Kuda is a library that can exchange requests / responses over JSON RPC protocol via the serial port.

# Feature

This library is designed to align with the interface of gorilla/rpc, which can be considered the definitive implementation of JSON-RPC in Go. If you are familiar with the gorilla/rpc interface, you can start using it right away.

# Sample

## A server side code

```go
package main

import (
	"context"
	"net/http"

	"github.com/gorilla/rpc/v2"
	"github.com/gorilla/rpc/v2/json2"

	"kuda"
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

func main() {
	s := rpc.NewServer()
	s.RegisterCodec(json2.NewCodec(), "")
	calculator := &Calculator{}
	s.RegisterService(calculator, "")

	srv := &kuda.Server{
		PortName: "/dev/ttyGS0",
	}
	srv.Serve(context.Background(), s)
}
```

## A client side code

```go
package main

import (
	"kuda"
	"log"
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
	added := 10
	add := 12

	client := kuda.Client{
		PortName: "COM9",
	}

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
```

# Note

This library has been tested by connecting the micro USB ports of a Windows machine and a Raspberry Pi Zero 2W. If you want to know whether it works in other environments, please verify it yourself.

# Author

- Yoshihiko Yamanaka (a.k.a. bamchoh)

# License

MIT License

Copyright 2025 bamchoh

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
