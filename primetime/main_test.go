package main

import (
	"fmt"
	"testing"
)

func TestProcess(t *testing.T) {

	type scenario struct {
		req  string
		resp Response
		err  error
	}

	scenarios := []scenario{
		{req: `{"method": "isPrime", "number": 0}`, resp: Response{Method: "isPrime", IsPrime: false}},
		{req: `{"method": "isPrime"}`, err: fmt.Errorf("validation error")},
		{req: `{"method": "isPrime", "number": 5}`, resp: Response{Method: "isPrime", IsPrime: true}},
		{req: `{"method": "isPrime", "number": 11}`, resp: Response{Method: "isPrime", IsPrime: true}},
		{req: `{"method": "isPrime", "number": 6}`, resp: Response{Method: "isPrime", IsPrime: false}},
		{req: `{"method": "isPrime", "number": 6.34}`, resp: Response{Method: "isPrime", IsPrime: false}},
		{req: `{"method": "sdfa", "number": 6.34}`, err: fmt.Errorf("some validation error")},
		{req: `{"method": "sdfa", "number": 6}`, err: fmt.Errorf("some validation error")},
		{req: `{"method": "isPrime", "number": "weew"}`, err: fmt.Errorf("some validation error")},
		{req: ``, err: fmt.Errorf("some validation error")},
	}

	for _, s := range scenarios {
		t.Run(s.req, func(t *testing.T) {
			resp, err := process(s.req)
			if s.err == nil && err != nil {
				t.Fatal("processing should not error")
			}
			if s.err != nil && err == nil {
				t.Fatal("processing should fail")
			}
			if err != nil {
				return
			}
			if resp.Method != s.resp.Method {
				t.Fatalf("response method doesn't match. expected: %s, got: %s", s.resp.Method, resp.Method)
			}
			if resp.IsPrime != s.resp.IsPrime {
				t.Fatalf("response isPrime doesn't match. expected: %v, got: %v", s.resp.IsPrime, resp.IsPrime)
			}
		})
	}
}
