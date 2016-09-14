package socketio

import (
	"errors"
	"fmt"
	"reflect"
)

type Caller interface {

	// build an array for decoding data into.
	GetArgs() []interface{}

	// Call the function passed arguments and return results.
	Call(so Socket, args []interface{}) []reflect.Value
}

type callFunc struct {
	Func       reflect.Value
	Args       []reflect.Type
	NeedSocket bool
}

func NewCaller(f interface{}) (Caller, error) {
	fv := reflect.ValueOf(f)
	if fv.Kind() != reflect.Func {
		return nil, fmt.Errorf("f is not func")
	}
	ft := fv.Type()
	if ft.NumIn() == 0 {
		return &callFunc{
			Func: fv,
		}, nil
	}
	args := make([]reflect.Type, ft.NumIn())
	for i, n := 0, ft.NumIn(); i < n; i++ {
		args[i] = ft.In(i)
	}
	needSocket := false
	if args[0].Name() == "Socket" {
		args = args[1:]
		needSocket = true
	}
	return &callFunc{
		Func:       fv,
		Args:       args,
		NeedSocket: needSocket,
	}, nil
}

// build an array for decoding data into.
func (c *callFunc) GetArgs() []interface{} {
	ret := make([]interface{}, len(c.Args))
	for i, argT := range c.Args {
		if argT.Kind() == reflect.Ptr {
			argT = argT.Elem()
		}
		v := reflect.New(argT)
		ret[i] = v.Interface()
	}
	return ret
}

// Call the function passed arguments and return results.
func (c *callFunc) Call(so Socket, args []interface{}) []reflect.Value {
	var a []reflect.Value
	diff := 0
	if c.NeedSocket {
		diff = 1
		a = make([]reflect.Value, len(args)+1)
		a[0] = reflect.ValueOf(so)
	} else {
		a = make([]reflect.Value, len(args))
	}

	if len(args) != len(c.Args) {
		return []reflect.Value{reflect.ValueOf([]interface{}{}), reflect.ValueOf(errors.New("Arguments do not match"))}
	}

	for i, arg := range args {
		v := reflect.ValueOf(arg)
		if c.Args[i].Kind() != reflect.Ptr {
			if v.IsValid() {
				v = v.Elem()
			} else {
				v = reflect.Zero(c.Args[i])
			}
		}
		a[i+diff] = v
	}

	return c.Func.Call(a)
}
