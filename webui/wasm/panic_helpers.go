package main

import "fmt"

func panicMessage(rec any) string {
	switch x := rec.(type) {
	case nil:
		return "panic: <nil>"
	case error:
		return x.Error()
	case fmt.Stringer:
		return x.String()
	default:
		return fmt.Sprint(x)
	}
}
