/*
*
Service is the core concept that is used to build up the engine.

The idea is that the implementation of workload/service actions and operation
should not care about endpoints, serialization, deserialization, of user input
*/
package engine

import (
	"context"
	"encoding/json"
	"reflect"
)

// Empty struct used as a quick shortcut to return No type
type Void struct{}

// The main action interface an action maps to a single
// function call that takes input I, and returns output O
type Action[I any, O any] interface {
	Do(ctx context.Context, input I) (output O, err error)
}

// ActionFn easily builds an action from a function
type ActionFn[I any, O any] func(ctx context.Context, input I) (output O, err error)

func (f ActionFn[I, O]) Do(ctx context.Context, input I) (output O, err error) {
	return f(ctx, input)
}

func (f ActionFn[I, O]) IntoService() Service {
	return IntoService[I, O](f)
}

// Service is a wrapper around an action and what implements invocation from
// input bytes and return back user output as Response object
type Service struct {
	input  reflect.Type
	method reflect.Value
}

func (s *Service) Call(ctx context.Context, input []byte) (output []byte, err error) {
	value := reflect.New(s.input)
	ptr := value.Interface()
	// we use json now, this might change in the future
	// or can be also modified per the request (input) content/type
	if err := json.Unmarshal(input, ptr); err != nil {
		return nil, err
	}

	elem := value.Elem()

	// we are 100% sure this is the method signature
	results := s.method.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		elem,
	})

	res := results[0].Interface()
	out, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	aErr := results[1].Interface()
	if aErr != nil {
		return nil, aErr.(error)
	}

	return out, nil
}

// IntoService converts an action into a service that can be called with a "Request"
func IntoService[I any, O any](action Action[I, O]) Service {
	input := reflect.TypeFor[I]()
	method := reflect.ValueOf(action).MethodByName("Do")

	return Service{
		input:  input,
		method: method,
	}
}
