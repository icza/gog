/*
Package gog contains general, generic extensions to the Go language,
requiring generics (introduced in Go 1.18).
*/
package gog

import "reflect"

// If returns vtrue if cond is true, vfalse otherwise.
//
// Useful to avoid an if statement when initializing variables, for example:
//
//	min := If(i > 0, i, 0)
func If[T any](cond bool, vtrue, vfalse T) T {
	if cond {
		return vtrue
	}
	return vfalse
}

// Coalesce returns the first value from the given parameters that is not evaluated to zero value
// Returns the last parameter value if all parameters are evaluated to zero value
//
// Useful when you want a chain of fallback options for your variable value
// For example:
//
//	hostVal := Coalesce(hostName, os.Getenv("HOST"), "localhost")
func Coalesce[T any](values ...T) (v T) {
	for _, v = range values {
		if v != nil && !reflect.ValueOf(v).IsZero() {
			break
		}
	}
	return
}

// Ptr returns a pointer to the passed value.
//
// Useful when you have a value and need a pointer, e.g.:
//
//	func f() string { return "foo" }
//
//	foo := struct{
//	    Bar *string
//	}{
//	    Bar: Ptr(f()),
//	}
func Ptr[T any](v T) *T {
	return &v
}

// Must takes 2 arguments, the second being an error.
// If err is not nil, Must panics. Else the first argument is returned.
//
// Useful when inputs to some function are provided in the source code,
// and you are sure they are valid (if not, it's OK to panic).
// For example:
//
//	t := Must(time.Parse("2006-01-02", "2022-04-20"))
func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// First returns the first argument.
// Useful when you want to use the first result of a function call that has more than one return values
// (e.g. in a composite literal or in a condition).
//
// For example:
//
//	func f() (i, j, k int, s string, f float64) { return }
//
//	p := image.Point{
//	    X: First(f()),
//	}
func First[T any](first T, _ ...any) T {
	return first
}

// Second returns the second argument.
// Useful when you want to use the second result of a function call that has more than one return values
// (e.g. in a composite literal or in a condition).
//
// For example:
//
//	func f() (i, j, k int, s string, f float64) { return }
//
//	p := image.Point{
//	    X: Second(f()),
//	}
func Second[T any](_ any, second T, _ ...any) T {
	return second
}

// Third returns the third argument.
// Useful when you want to use the third result of a function call that has more than one return values
// (e.g. in a composite literal or in a condition).
//
// For example:
//
//	func f() (i, j, k int, s string, f float64) { return }
//
//	p := image.Point{
//	    X: Third(f()),
//	}
func Third[T any](_, _ any, third T, _ ...any) T {
	return third
}
