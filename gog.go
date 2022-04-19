package gog

// If returns vtrue if cond is true, vfalse otherwise.
//
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
