package main

import "math"

func intMod(a, b int) int {
	return a % b
}

func floatMod(a, b float64) float64 {
	return math.Mod(a, b)
}

func runMod() (number, error) {
	a, err := left()
	if err != nil {
		return number{}, err
	}
	b, err := denominatorRight()
	if err != nil {
		return number{}, err
	}
	return castMod(a, b), nil
}

func denominatorRight() (number, error) {
	n, err := right()
	if !n.Bool() && err == nil {
		err = ErrZeroDivision
	}
	return n, err
}

func chooseMod(a, b number) number {
	if a.isFloat {
		a.f = intMod(a.f, b.f)
	} else {
		a.i = floatMod(a.i, b.i)
	}
	return a
}

func castMod(a, b number) number {
	if a.isFloat != b.isFloat {
		if !a.isFloat {
			a = number{f: float64(a.i), isFloat: true}
		} else {
			b = number{f: float64(b.i), isFloat: true}
		}
	}
	return chooseMod(a, b)
}

func newDivModOpForModulo() {
	return divModOp(castMod)
}
