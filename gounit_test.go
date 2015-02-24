package gounit

import (
	"fmt"
	"testing"
)

func TestBlankFixtureDescriptionRegistration(t *testing.T) {
	spy := NewSpyT()

	f := NewFixture("", spy)
	f.Test("mute point", func() {})
	f.Run()

	if ok, message := So(spy.failed, ShouldBeTrue); !ok {
		t.Error(message)
	}
}

func TestNoTestCases(t *testing.T) {
	spy := NewSpyT()

	f := NewFixture("A", spy)
	f.Setup(func() {})
	f.Teardown(func() {})
	f.Run()

	if ok, message := So(spy.skipped, ShouldBeTrue); !ok {
		t.Error(message)
	}
}

func TestBlankTestCaseRegistration(t *testing.T) {
	spy := NewSpyT()

	f := NewFixture("A", spy)
	f.Test("", func() {})
	f.Run()

	if ok, message := So(spy.failed, ShouldBeTrue); !ok {
		t.Error("\n" + message)
	}
}

func TestDuplicateTestCaseRegistration(t *testing.T) {
	spy := NewSpyT()

	f := NewFixture("A", spy)
	f.Test("B", func() {})
	f.Test("B", func() {})
	f.Run()

	if ok, message := So(spy.failed, ShouldBeTrue); !ok {
		t.Error("\n" + message)
	}
}

func TestSkipNewFixture(t *testing.T) {
	spy := NewSpyT()

	setup, teardown := 0, 0
	b1, b2, b3, b4, b5, b6 := false, false, false, false, false, false
	f := SkipNewFixture("A", spy)
	f.Setup(func() { setup++ })
	f.Teardown(func() { teardown++ })
	f.Test("B1", func() { b1 = true })
	f.SkipTest("B2", func() { b2 = true })
	f.FocusTest("B3", func() { b3 = true })
	f.GoTest("B4", func(done func()) { defer done(); b4 = true })
	f.SkipGoTest("B5", func(done func()) { defer done(); b5 = true })
	f.FocusGoTest("B6", func(done func()) { defer done(); b6 = true })
	f.Run()

	if ok, message := So(spy.skipped, ShouldBeTrue); !ok {
		t.Error(message)
	}

	if ok, message := So(setup, ShouldEqual, 0); !ok {
		t.Error("\n" + message)
	}
	if ok, message := So(teardown, ShouldEqual, 0); !ok {
		t.Error("\n" + message)
	}
	if ok, message := So(b1, ShouldBeFalse); !ok {
		t.Error("\n" + message)
	}
	if ok, message := So(b2, ShouldBeFalse); !ok {
		t.Error("\n" + message)
	}
	if ok, message := So(b3, ShouldBeFalse); !ok {
		t.Error("\n" + message)
	}
	if ok, message := So(b4, ShouldBeFalse); !ok {
		t.Error("\n" + message)
	}
	if ok, message := So(b5, ShouldBeFalse); !ok {
		t.Error("\n" + message)
	}
	if ok, message := So(b6, ShouldBeFalse); !ok {
		t.Error("\n" + message)
	}
}

func TestPassingTests(t *testing.T) {
	spy := NewSpyT()

	b1, b2 := false, false

	f := NewFixture("A", spy)
	f.Test("B1", func() {
		b1 = true
	})
	f.GoTest("B2", func(done func()) {
		go func() {
			b2 = true
			done()
		}()
	})
	f.Run()

	if ok, message := So(b1, ShouldBeTrue); !ok {
		t.Error("\n" + message)
	}
	if ok, message := So(b2, ShouldBeTrue); !ok {
		t.Error("\n" + message)
	}
}

func TestFailingTest(t *testing.T) {
	spy := NewSpyT()

	b1 := false

	f := NewFixture("A", spy)
	f.Test("B1", func() {
		b1 = true
		f.So("For the sake of the test I'll say this should be false", b1, ShouldBeFalse)
	})
	f.Run()

	if ok, message := So(spy.failed, ShouldBeTrue); !ok {
		t.Error("\n" + message)
	}

	if ok, message := So(b1, ShouldBeTrue); !ok {
		t.Error("\n" + message)
	}
}

func TestFailingGoTest(t *testing.T) {
	spy := NewSpyT()

	b1 := false

	f := NewFixture("A", spy)
	f.GoTest("B1", func(done func()) {
		go func() {
			b1 = true
			f.So("For the sake of the test, I'll say this should be false", b1, ShouldBeFalse)
			done()
		}()
	})
	f.Run()

	if ok, message := So(spy.failed, ShouldBeTrue); !ok {
		t.Error("\n" + message)
	}

	if ok, message := So(b1, ShouldBeTrue); !ok {
		t.Error("\n" + message)
	}
}

func TestPanickingTest(t *testing.T) {
	spy := NewSpyT()

	f := NewFixture("A", spy)

	f.Test("B1", func() {
		panic("GOPHERS!")
	})
	f.SkipGoTest("B2", func(done func()) {
		panic("GOPHERS!")
		done()
	})
	f.Run()

	if ok, message := So(spy.failed, ShouldBeTrue); !ok {
		t.Error(message)
	}

	f = NewFixture("A", spy)

	f.SkipGoTest("B2", func(done func()) {
		panic("GOPHERS!")
		done()
	})
	f.Run()

	if ok, message := So(spy.failed, ShouldBeTrue); !ok {
		t.Error(message)
	}
}

func TestSkippedTests(t *testing.T) {
	spy := NewSpyT()

	skip1, hi, skip2 := false, false, false

	f := NewFixture("A", spy)
	f.SkipTest("skip1", func() {
		skip1 = true // shouldn't happen
	})
	f.Test("hi", func() {
		hi = true
	})
	f.SkipGoTest("skip2", func(done func()) {
		go func() {
			skip2 = true // shouldn't happen
			done()
		}()
	})
	f.Run()

	if !hi {
		t.Error("Active test was skipped when it should have beed executed.")
	}
	if skip1 {
		t.Error("Skipped test (skip1) was run when it should have been skipped.")
	}
	if skip2 {
		t.Error("Skipped test (skip2) was run when it should have been skipped.")
	}
}

func TestFocusedTests(t *testing.T) {
	spy := NewSpyT()

	b1, b2, b3, b4 := false, false, false, false

	f := NewFixture("A", spy)
	f.Test("B1", func() { b1 = true })
	f.FocusTest("B2", func() { b2 = true })
	f.GoTest("B3", func(done func()) { b3 = true; done() })
	f.FocusGoTest("B4", func(done func()) { b4 = true; done() })
	f.Run()

	if ok, message := So(b1, ShouldBeFalse); !ok {
		t.Error(message)
	}

	if ok, message := So(b2, ShouldBeTrue); !ok {
		t.Error(message)
	}

	if ok, message := So(b3, ShouldBeFalse); !ok {
		t.Error(message)
	}

	if ok, message := So(b4, ShouldBeTrue); !ok {
		t.Error(message)
	}
}

func TestSkippedSoAssertion(t *testing.T) {
	spy := NewSpyT()

	f := NewFixture("A", spy)
	f.Test("B1", func() {
		f.SkipSo("if this assertion runs, the overall test will fail", false, ShouldBeTrue)
	})
	f.GoTest("B2", func(done func()) {
		defer done()
		f.SkipSo("if this assertion runs, the overall test will fail", false, ShouldBeTrue)
	})
	f.Run()

	if ok, message := So(spy.failed, ShouldBeFalse); !ok {
		t.Error(message)
	}
}

func TestSetup(t *testing.T) {
	spy := NewSpyT()

	setup := 0
	f := NewFixture("A", spy)
	f.Setup(func() { setup++ })
	f.Test("B1", func() {})
	f.Test("B2", func() { panic("GOPHERS!") })
	f.GoTest("B3", func(done func()) { defer done() })
	f.GoTest("B4", func(done func()) { defer done(); panic("GOPHERS!") })
	f.Run()

	if ok, message := So(setup, ShouldEqual, 4); !ok {
		t.Errorf("\n" + message)
	}
}

func TestSetupPanics(t *testing.T) {
	spy := NewSpyT()
	f := NewFixture("A", spy)
	f.Setup(func() { panic("GOPHERS!") })
	f.Test("B1", func() {})
	f.Run()

	if ok, message := So(spy.failed, ShouldBeTrue); !ok {
		t.Errorf(message)
	}
}

func TestTeardown(t *testing.T) {
	spy := NewSpyT()

	teardown := 0

	f := NewFixture("A", spy)
	f.Teardown(func() { teardown++ })
	f.Test("B1", func() {})
	f.Test("B2", func() { panic("GOPHERS!") })
	f.GoTest("B3", func(done func()) { defer done() })
	f.GoTest("B4", func(done func()) { defer done(); panic("GOPHERS!") })
	f.Run()

	if ok, message := So(teardown, ShouldEqual, 4); !ok {
		t.Errorf("\n" + message)
	}
}

func TestTeardownPanics(t *testing.T) {
	spy := NewSpyT()
	f := NewFixture("A", spy)
	f.Test("B1", func() {})
	f.Teardown(func() { panic("GOPHERS!") })
	f.Run()

	if ok, message := So(spy.failed, ShouldBeTrue); !ok {
		t.Errorf(message)
	}
}

func TestFixtureDisabledAfterRun(t *testing.T) {
	spy := NewSpyT()

	f := NewFixture("A", spy)
	f.Test("This runs", func() {})
	f.Run()

	if ok, message := So(spy.failed, ShouldBeFalse); !ok {
		t.Error(message)
	}

	f.Test("This doesn't run", func() { f.So("really, this doesn't run", true, ShouldBeFalse) })
	f.Run()

	if ok, message := So(spy.failed, ShouldBeFalse); !ok {
		t.Error(message)
	}
}

//////////////////////////////////////////////////////////////////////////////

// spyT is a stand-in for a *testing.T, at least as far as the gounit package is concerned.
type spyT struct {
	failed  bool
	skipped bool
	log     string
}

func NewSpyT() *spyT                       { return &spyT{} }
func (self *spyT) Fail()                   { self.failed = true }
func (self *spyT) Failed() bool            { return self.failed }
func (self *spyT) SkipNow()                { self.skipped = true }
func (self *spyT) Log(args ...interface{}) { self.log = fmt.Sprint(args...) }

//////////////////////////////////////////////////////////////////////////////
