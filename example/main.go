package tr

//go:generate go run github.com/GeorgeEngland/typedtemporal/cmd/typedtemporal -debug

import (
	"github.com/GeorgeEngland/typedtemporal"
	"go.temporal.io/sdk/workflow"
)

var workflows = []typedtemporal.Workflow{
	{Name: "sayHello", Description: "asd", Func: HelloWorkflow},
	// {Name: "sayHelloInLine", Description: "asd",
	// 	Func: func(w workflow.Context, s string) (Response, error) {
	// 		return Response{Res: ""}, nil
	// 	}},
	{Name: "sayHello2", Description: "asd", Func: HelloWorkflow},
}

func HelloWorkflow(ctx workflow.Context, params string) (Response, error) {
	return Response{Res: "hello: " + params}, nil
}
