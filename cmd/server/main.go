package main

import (
	"context"
	"net/http"

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
