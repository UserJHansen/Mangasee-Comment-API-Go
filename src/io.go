package main

import (
	"fmt"
)

func Println(a ...interface{}) {
	_, _ = fmt.Fprintln(out, a...)
}

func Printf(format string, a ...interface{}) {
	_, _ = fmt.Fprintf(out, format, a...)
}
