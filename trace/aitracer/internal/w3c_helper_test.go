package internal

import (
	"fmt"
	"testing"
)

func TestW3C(t *testing.T) {
	s := SimpleW3CFormatParser{}
	parent, err := s.ParseTraceParent("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	fmt.Printf("%+v\n", parent)
	fmt.Printf("err=%+v\n", err)

	state, err := s.ParseTraceState("rojo=00f067aa0ba902b7,congo=t61rcWkgMzE")
	fmt.Printf("%+v\n", state)
	fmt.Printf("err=%+v\n", err)
}

func TestW3C2(t *testing.T) {
	s := SimpleW3CFormatParser{}
	parent, err := s.ParseTraceParent("00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-")
	fmt.Printf("%+v\n", parent)
	fmt.Printf("err=%+v\n", err)

	state, err := s.ParseTraceState("k0=,,,,,k1=v1,k2=v2,,,,,")
	fmt.Printf("%+v\n", state)
	fmt.Printf("err=%+v\n", err)
}

func TestW3C3(t *testing.T) {
	s := SimpleW3CFormatParser{}
	parent, err := s.ParseTraceParent("")
	fmt.Printf("%+v\n", parent)
	fmt.Printf("err=%+v\n", err)

	state, err := s.ParseTraceState("")
	fmt.Printf("%+v\n", state)
	fmt.Printf("err=%+v\n", err)
}

func TestW3C4(t *testing.T) {
	s := SimpleW3CFormatParser{}
	parent, err := s.ParseTraceParent("00-00000000000000000000000000000001-0000000000000000-01")
	fmt.Printf("%+v\n", parent)
	fmt.Printf("err=%+v\n", err)

	state, err := s.ParseTraceState("k0=v0,k1=v1=v2")
	fmt.Printf("%+v\n", state)
	fmt.Printf("err=%+v\n", err)
}
