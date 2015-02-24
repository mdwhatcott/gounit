package gounit

import (
	"strconv"
	"testing"

	. "github.com/mdwhatcott/gounit"
)

func TestIncrementDecrement(t *testing.T) {
	var x int

	f := NewFixture("Addition and subtraction", t)
	defer f.Run()

	f.Setup(func() {
		x = 0
	})

	f.GoTest("Addition should work", func(done func()) {
		go func() {
			x++
			f.So("The number should increment", x, ShouldEqual, 1)
			done()
		}()
	})

	f.Test("Subtraction should work", func() {
		x--
		f.So("The number should be decremented", x, ShouldEqual, -1)
	})

	f.Teardown(func() {
		x = 0
	})
}

func TestTable(t *testing.T) {
	fixture := NewFixture("Table-driven testing!", t)
	defer fixture.Run()

	for i, testCase := range []int{0, 1, 2, 3, 4, 5} {
		fixture.Test("TestCase #"+strconv.Itoa(i), func() {
			fixture.So("The index and value should match", i, ShouldEqual, testCase)
		})
	}
}
