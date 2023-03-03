package gog_test

import (
	"fmt"
	"time"

	"github.com/icza/gog"
)

func ExampleOpCache() {
	type Point struct {
		X, Y    int
		Counter int
	}

	// Existing function we want to add caching for:
	counter := 0
	GetPoint := func(x, y int) (*Point, error) {
		counter++
		return &Point{X: x, Y: y, Counter: counter}, nil
	}

	var getPointCache = gog.NewOpCache[*Point](gog.OpCacheConfig{ResultExpiration: 100 * time.Millisecond})

	// Function to use which utilizes getPointCache (has identical signature with GetPoint):
	GetPointFast := func(x, y int) (*Point, error) {
		return getPointCache.Get(
			fmt.Sprint(x, y), // Key of all params
			func() (*Point, error) { return GetPoint(x, y) },
		)
	}

	p, err := GetPointFast(1, 2) // This will call GetPoint()
	fmt.Printf("%+v %v\n", p, err)
	p, err = GetPointFast(1, 2) // This will come from the cache
	fmt.Printf("%+v %v\n", p, err)

	time.Sleep(110 * time.Millisecond)
	p, err = GetPointFast(1, 2) // Cache expired, will call GetPoint() again
	fmt.Printf("%+v %v\n", p, err)

	// Output:
	// &{X:1 Y:2 Counter:1} <nil>
	// &{X:1 Y:2 Counter:1} <nil>
	// &{X:1 Y:2 Counter:2} <nil>
}

func ExampleOpCache_multi_return() {
	type Point struct {
		X, Y    int
		Counter int
	}

	// Existing function we want to add caching for:
	counter := 0
	GetPoint := func(x, y int) (*Point, int, error) {
		counter++
		return &Point{X: x, Y: 2 * x, Counter: counter}, counter * 10, fmt.Errorf("test_error_%d", counter)
	}

	// this type wraps the multiple return types of GetPoint():
	type multiResults struct {
		p *Point
		n int
	}
	var getPointCache = gog.NewOpCache[multiResults](gog.OpCacheConfig{ResultExpiration: 100 * time.Millisecond})

	// Function to use which utilizes getPointCache (has identical signature with GetPoint):
	GetPointFast := func(x, y int) (*Point, int, error) {
		mr, err := getPointCache.Get(
			fmt.Sprint(x, y), // Key of all params
			func() (multiResults, error) {
				p, n, err := GetPoint(x, y)
				return multiResults{p, n}, err // packing multiple results
			},
		)
		return mr.p, mr.n, err // Unpacking multiple results
	}

	p, n, err := GetPointFast(1, 2) // This will call GetPoint()
	fmt.Printf("%+v %d %v\n", p, n, err)
	p, n, err = GetPointFast(1, 2) // This will come from the cache
	fmt.Printf("%+v %d %v\n", p, n, err)

	time.Sleep(110 * time.Millisecond)
	p, n, err = GetPointFast(1, 2) // Cache expired, will call GetPoint() again
	fmt.Printf("%+v %d %v\n", p, n, err)

	// Output:
	// &{X:1 Y:2 Counter:1} 10 test_error_1
	// &{X:1 Y:2 Counter:1} 10 test_error_1
	// &{X:1 Y:2 Counter:2} 20 test_error_2
}
