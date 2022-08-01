package main

import "fmt"

func clos(s string, i int) func() {
	return func() {
		fmt.Println(s, "=", i) // this closure captures s and i
		i++                    // it increments i each time it runs
	}
}

func main() {
	i := 42             // Each call to clos returns a new closure
	clos("i", i)()      // that has captured
	clos("i", i)()      // its own variables.
	f := clos("\tf", 5) // Therefore f and g
	g := clos("g", 23)  // will increment their
	f()                 // own integer variables
	f()                 // if run repeatedly.
	g()
	f()
	g()
	g()
	f()
}
