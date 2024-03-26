/*
Package slicesx provides generic slice utility functions.
*/
package slicesx

// Props returns a slice of a property, which is accessed using the given getter.
func Props[V, P any](c []V, getter func(v V) P) []P {
	props := make([]P, len(c))
	for i, v := range c {
		props[i] = getter(v)
	}
	return props
}

// PropMap returns a map, mapping from a property, which is accessed using the given getter.
func PropMap[V any, P comparable](c []V, getter func(v V) P) map[P]V {
	m := make(map[P]V, len(c))
	for _, v := range c {
		m[getter(v)] = v
	}
	return m
}

// PropsMap returns a map, mapping from a property, which is accessed using the given getter.
// It is allowed / normal that multiple elements have the same property value, so map values are slices collecting those elements.
func PropsMap[V any, P comparable](c []V, getter func(v V) P) map[P][]V {
	m := make(map[P][]V)
	for _, v := range c {
		p := getter(v)
		m[p] = append(m[p], v)
	}
	return m
}

// Filter returns a new slice holding only the filtered elements.
func Filter[V any](c []V, f func(v V) bool) []V {
	var out []V
	for _, v := range c {
		if f(v) {
			out = append(out, v)
		}
	}
	return out
}
