package gog

import (
	"errors"
	"testing"
)

func TestIf(t *testing.T) {
	{
		i1, i2 := 1, 2
		exp, got := i1, If(true, i1, i2)
		if got != exp {
			t.Errorf("[int] Expected %d, got: %d", exp, got)
		}
		exp, got = i2, If(false, i1, i2)
		if got != exp {
			t.Errorf("[int] Expected %d, got: %d", exp, got)
		}
	}

	{
		s1, s2 := "first", "second"
		exp, got := s1, If(true, s1, s2)
		if got != exp {
			t.Errorf("[string] Expected %s, got: %s", exp, got)
		}
		exp, got = s2, If(false, s1, s2)
		if got != exp {
			t.Errorf("[string] Expected %s, got: %s", exp, got)
		}
	}
}

func TestPtr(t *testing.T) {
	s := "a"
	sp := Ptr(s)
	if *sp != s {
		t.Errorf("Ptr[string] failed")
	}

	i := 2
	ip := Ptr(i)
	if *ip != i {
		t.Errorf("Ptr[int] failed")
	}
}

func TestMust(t *testing.T) {
	i := 1
	if got := Must(i, nil); got != i {
		t.Errorf("Must[int] failed")
	}

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic")
			}
		}()
		Must(i, errors.New("test")) // Expecting panic
		t.Error("Not expected to reach this")
	}()
}

func manyResults() (i, j, k int, s string, f float64) {
	return 1, 2, 3, "four", 5.0
}

func TestFirst(t *testing.T) {
	exp, got := 1, First(manyResults())
	if got != exp {
		t.Errorf("Expected %d, got: %d", exp, got)
	}
}

func TestSecond(t *testing.T) {
	exp, got := 2, Second(manyResults())
	if got != exp {
		t.Errorf("Expected %d, got: %d", exp, got)
	}
}
func TestThird(t *testing.T) {
	exp, got := 3, Third(manyResults())
	if got != exp {
		t.Errorf("Expected %d, got: %d", exp, got)
	}
}

func TestCoalesce(t *testing.T) {
	p1, p2 := Ptr(1), Ptr(2)

	cases := []struct {
		name     string
		exp, got any
	}{
		{
			"strings",
			"1", Coalesce("", "1", "2"),
		},
		{
			"strings first",
			"1", Coalesce("1", "2", "3"),
		},
		{
			"strings last",
			"1", Coalesce("", "", "1"),
		},
		{
			"strings all zero",
			"", Coalesce("", "", ""),
		},
		{
			"strings no args",
			"", Coalesce[string](),
		},
		{
			"ints",
			1, Coalesce(0, 1, 2, 3),
		},
		{
			"ints first",
			1, Coalesce(1, 2, 3),
		},
		{
			"ints last",
			1, Coalesce(0, 0, 0, 0, 1),
		},
		{
			"ints all zero",
			0, Coalesce(0, 0, 0, 0),
		},
		{
			"ints no args",
			0, Coalesce[int](),
		},
		{
			"pointers",
			p1, Coalesce(nil, p1, p2),
		},
		{
			"pointers first",
			p1, Coalesce(p1, p2),
		},
		{
			"pointers last",
			p1, Coalesce(nil, nil, p1),
		},
		{
			"pointers all zero",
			(*int)(nil), Coalesce[*int](nil, nil, nil),
		},
		{
			"pointers no args",
			(*int)(nil), Coalesce[*int](),
		},
	}

	for _, c := range cases {
		if c.exp != c.got {
			t.Errorf("[%s] Expected: %v, got: %v", c.name, c.exp, c.got)
		}
	}

}
