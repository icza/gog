package gog

// If returns vtrue if cond is true, vfalse otherwise.
func If[T any](cond bool, vtrue, vfalse T) T {
	if cond {
		return vtrue
	}
	return vfalse
}

// Ptr returns a pointer to the passed value.
func Ptr[T any](v T) *T {
	return &v
}

// Must takes 2 arguments, the second being an error.
// If err is not nil, Must panics. Else the first argument is returned.
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
//   func f() (i, j, k int, s string, f float64) { return }
//   p := image.Point{
//       X: First(f()),
//   }
func First[T any](first T, _ ...any) T {
	return first
}

// Second returns the second argument.
// Useful when you want to use the second result of a function call that has more than one return values
// (e.g. in a composite literal or in a condition).
//
// For example:
//   func f() (i, j, k int, s string, f float64) { return }
//   p := image.Point{
//       X: Second(f()),
//   }
func Second[T any](_ any, second T, _ ...any) T {
	return second
}

// Third returns the third argument.
// Useful when you want to use the third result of a function call that has more than one return values
// (e.g. in a composite literal or in a condition).
//
// For example:
//   func f() (i, j, k int, s string, f float64) { return }
//   p := image.Point{
//       X: Third(f()),
//   }
func Third[T any](_, _ any, third T, _ ...any) T {
	return third
}
