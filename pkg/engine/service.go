/*
*
Service is the core concept that is used to build up the engine.

The idea is that the implementation of workload/service actions and operation
should not care about endpoints, serialization, deserialization, of user input
*/
package engine

import (
	"encoding/json"
	"reflect"
)

// Empty struct used as a quick shortcut to return No type
type Void struct{}

func (v Void) MarshalJSON() ([]byte, error) {
	return []byte("null"), nil
}

// The main action interface an action maps to a single
// function call that takes input I, and returns output O
type Action[I any, O any] interface {
	Do(ctx Context, input I) (output O, err error)
}

type IntoService interface {
	Into(flags ...ServiceFlag) Service
}

// ActionFn easily builds an action from a function
type ActionFn[I any, O any] func(ctx Context, input I) (output O, err error)

func NewAction[I any, O any](action func(ctx Context, input I) (output O, err error)) ActionFn[I, O] {
	return ActionFn[I, O](action)
}

func (f ActionFn[I, O]) Do(ctx Context, input I) (output O, err error) {
	return f(ctx, input)
}

func (f ActionFn[I, O]) Into(flags ...ServiceFlag) Service {
	return ActionIntoService[I, O](f, flags...)
}

// Service is a wrapper around an action and what implements invocation from
// input bytes and return back user output as Response object
type Service struct {
	flags  ServiceFlag
	input  reflect.Type
	method reflect.Value
}

func (s *Service) Call(ctx Context, input []byte) (output []byte, err error) {
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
func ActionIntoService[I any, O any](action Action[I, O], flags ...ServiceFlag) Service {
	input := reflect.TypeFor[I]()
	method := reflect.ValueOf(action).MethodByName("Do")

	var f ServiceFlag
	if len(flags) == 1 {
		f = flags[0]
	} else if len(flags) > 1 {
		panic("flags must be provided only once. use bitwise or to or them")
	}

	return Service{
		flags:  f,
		input:  input,
		method: method,
	}
}

type ServiceFlag uint8

const (
	// A call to this service only if no object exist with this id. It also implies
	// exclusive access
	ServiceObjectMustNotExist ServiceFlag = 1 << iota
	// A call to this service is only possible if object exists with this id. Default
	// if ObjectMustNotExist is not set
	// ServiceObjectMustExists
	// An exclusive service means the function must have an exclusive
	// access to the resource when executing this service.
	ServiceExclusive
)

func (s ServiceFlag) Is(f ServiceFlag) bool {
	return s&f == f
}
