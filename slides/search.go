package main

import "sort"

func Search(n int, f func(int) bool) int // from package "sort"

// findMeAnInt returns the index of the first number >=k in the sorted array a.
func findMeAnInt(a []int, k int) int { // HL
	/*
	 * a and k are available in the "environment" within the function
	 * findMeAnInt.  These are different variables on each function call.
	 *
	 * The second argument of sort.Search is an anonymous function that
	 * receives i as an argument and captures a and k from the environment.
	 */
	return sort.Search(len(a),
		func(i int) bool { // HL
			return k < a[i] // HL
		}) // HL
}
