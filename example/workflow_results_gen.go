package tr

import (
	"context"

	"go.temporal.io/sdk/client"
)



func ExecuteSayHelloWorkflow(ctx context.Context,
	c client.Client,
	options client.StartWorkflowOptions,
	input *string) (sayHelloRun, error) {
	r, err := c.ExecuteWorkflow(ctx, options, HelloWorkflow, input)
	if err != nil {
		return nil, err
	}

	return &sayHelloRunImpl{
		c:   c,
		run: r,
	}, nil

}

type temporalsayHelloWorkflowRun interface {
	Get(context.Context, interface{}) error
}

type sayHelloRun interface {
	Get(context.Context, *Response) error
}

type sayHelloRunImpl struct {
	c   client.Client
	run temporalsayHelloWorkflowRun
}

func (s *sayHelloRunImpl) Get(ctx context.Context, result *Response) error {
	return s.run.Get(ctx, result)
}



func ExecuteSayHello2Workflow(ctx context.Context,
	c client.Client,
	options client.StartWorkflowOptions,
	input *string) (sayHello2Run, error) {
	r, err := c.ExecuteWorkflow(ctx, options, HelloWorkflow, input)
	if err != nil {
		return nil, err
	}

	return &sayHello2RunImpl{
		c:   c,
		run: r,
	}, nil

}

type temporalsayHello2WorkflowRun interface {
	Get(context.Context, interface{}) error
}

type sayHello2Run interface {
	Get(context.Context, *Response) error
}

type sayHello2RunImpl struct {
	c   client.Client
	run temporalsayHello2WorkflowRun
}

func (s *sayHello2RunImpl) Get(ctx context.Context, result *Response) error {
	return s.run.Get(ctx, result)
}


