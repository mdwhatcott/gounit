// Package gounit implements xunit for Go (along with some other goodies).
//
// http://en.wikipedia.org/wiki/XUnit
//
// (No attempt has yet been made to produce XUnit-style XML output.)
package gounit

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/smartystreets/assertions"
)

// T contains the methods we use on the testing.T that is passed into the fixture.
// Using this interface instead of the testing.T directly allows for easier
// verification of correct behavior via automated testing.
type T interface {
	Fail()
	SkipNow()
	Log(...interface{})
}

// A simple xunit-style test fixture. Call NewFixture to create one.
type Fixture struct {
	t      T
	waiter *sync.WaitGroup

	frozen  bool // frozen prevents setup, teardown, and tests from being registered.
	spoiled bool // spoiled marks the whole fixture as failed.

	setup    func()
	teardown func()

	tests   map[string]func(func())
	focused map[string]struct{}
	skipped map[string]struct{}

	output *bytes.Buffer
}

// NewFixture creates a new test fixture. Now you can call the attached
// methods to register and run test cases and optional setup and teardown
// functions. Because these methods return their receiver you have the option
// to chain the method calls if you like that sort of thing (I know I do).
func NewFixture(description string, t T) *Fixture {
	return &Fixture{
		t:      t,
		waiter: new(sync.WaitGroup),

		setup:    func() {},
		teardown: func() {},

		tests:   make(map[string]func(func())),
		focused: make(map[string]struct{}),
		skipped: make(map[string]struct{}),

		output:  bytes.NewBufferString(description + "\n"),
		spoiled: len(description) == 0,
	}
}

func SkipNewFixture(description string, t T) *Fixture {
	return &Fixture{
		t:      t,
		frozen: true,
	}
}

// Setup registers a function to be run before any and all test cases.
// Subsequent calls to this function overwrite the previously registered
// setup function.
func (self *Fixture) Setup(action func()) {
	if self.frozen {
		return
	}
	self.setup = action
}

// Teardown registers a function to be run after any and all test cases,
// even when test cases panic. Subsequent calls to this function
// overwrite the previously registered teardown function.
func (self *Fixture) Teardown(action func()) {
	if self.frozen {
		return
	}
	self.teardown = action
}

// Test registers a test case, to be run after any registered setup and
// before any registered teardown. Test cases must have unique descriptions
// within the context of a Fixture.
func (self *Fixture) Test(description string, action func()) {
	if self.frozen {
		return
	}
	self.validate(description)

	self.tests[description] = func(done func()) {
		defer done()
		action()
	}
}

// SkipTest registers a test case to be logged in test output but it
// will not be executed. A call of this function is meant to aid
// debugging and development and should be replaced with a call to the
// Test function as soon as possible.
func (self *Fixture) SkipTest(description string, action func()) {
	if self.frozen {
		return
	}
	self.validate(description)
	self.tests[description] = nil
	self.skipped[description] = struct{}{}
}

// FocusTest registers a test to be run instead of any other tests not
// registered with this function. A call of this function is meant to
// aid debugging and development and should be replaced with a call to
// the Test function as soon as possible.
func (self *Fixture) FocusTest(description string, action func()) {
	if self.frozen {
		return
	}
	self.validate(description)
	self.focused[description] = struct{}{}
	self.Test(description, action)
}

// GoTest registers a test case, to be run after any registered setup and
// before any registered teardown. Use GoTest in favor of the Test function
// when your action launches another goroutine, thus relenting flow of
// execution from your code back to this library. To avoid the teardown or
// additional test cases running away before your code finishes, call the
// done func() passed into the action as its last instruction. Test cases
// must have unique descriptions within the context of a Fixture.
func (self *Fixture) GoTest(description string, action func(func())) {
	if self.frozen {
		return
	}
	self.validate(description)
	self.tests[description] = action
}

// SkipGoTest registers a test case to be logged in test output but it
// will not be executed. It is analogous to SkipTest and is meant for
// concurrent scenarios. A call of this function is meant to aid debugging
// and development and should be replaced with a call to the Test function
// as soon as possible.
func (self *Fixture) SkipGoTest(description string, action func(func())) {
	if self.frozen {
		return
	}
	self.validate(description)
	self.tests[description] = nil
	self.skipped[description] = struct{}{}
}

// FocusGoTest registers a test to be run instead of any other tests not
// registered with this function. It is analogous to FocusTest and is meant
// for concurrent scenarios. A call of this function is meant to aid debugging
// and development and should be replaced with a call to the Test function as
// soon as possible.
func (self *Fixture) FocusGoTest(description string, action func(func())) {
	if self.frozen {
		return
	}
	self.validate(description)
	self.focused[description] = struct{}{}
	self.GoTest(description, action)
}

func (self *Fixture) validate(description string) {
	if len(description) == 0 {
		self.spoiled = true
		self.Log("Test description must be non-blank.\n")
	} else if _, found := self.tests[description]; found {
		self.spoiled = true
		self.Logf(
			"Description conflict: action already registered with this description: '%s'\n",
			description)
	}
}

// Run iterates all test cases performing the following steps:
// - If registered, run the setup function.
// - Run the test case.
// - If registered, run the teardown function.
func (self *Fixture) Run() {
	defer self.dump()

	if self.frozen || len(self.tests) == 0 {
		self.t.SkipNow() // calls runtime.Goexit(), killing the current goroutine
	} else if self.spoiled {
		self.t.Fail()
	} else {
		self.runAll()
	}
}

func (self *Fixture) dump() {
	self.t.Log(self.output.String())
}

func (self *Fixture) runAll() {
	self.frozen = true

	for description, test := range self.tests {
		self.runOne(description, test)
	}
}

func (self *Fixture) runOne(description string, test func(func())) {
	if len(self.focused) > 0 {
		if _, focus := self.focused[description]; focus {
			self.execute(" -> <FOCUSED> ", description, test)
		} else {
			self.Logf(" -> (skipped) \"%s\"\n", description)
		}
	} else if _, skip := self.skipped[description]; skip {
		self.Logf(" -> (skipped) \"%s\"\n", description)
	} else {
		self.execute(" -> ", description, test)
	}
}

func (self *Fixture) execute(prefix, description string, test func(func())) {
	defer self.recover() // recovers panic in teardown
	defer self.teardown()
	defer self.recover() // recovers panic in setup
	self.setup()
	self.Logf("%s\"%s\"\n", prefix, description)
	self.waiter.Add(1)
	test(func() { defer self.recoverDone() }) // recovers panic in test
	self.waiter.Wait()
}

func (self *Fixture) recoverDone() {
	self.recover()
	self.waiter.Done()
}

func (self *Fixture) recover() {
	if r := recover(); r != nil {
		self.t.Fail()
		self.Log(self.formatPanic(fmt.Sprint(r)))
	}
}

func (self *Fixture) formatPanic(recovered string) string {
	_, file, line, _ := runtime.Caller(4)
	fileInfo := file + ":" + strconv.Itoa(line)
	title := "PANIC: [" + recovered + "]"
	divider := strings.Repeat("*", max(len(fileInfo), len(title)))
	return "\n\n  " + divider + "\n\n  " +
		title + "\n\n  " +
		fileInfo + "\n\n  " +
		divider + "\n"
}

// This method stands in as a 'So' call with a required description--
// (a-la-`github.com/smartystreets/goconvey/convey/assertions.So`)
func (self *Fixture) So(description string, actual interface{}, so func(actual interface{}, expected ...interface{}) string, expected ...interface{}) {
	ok, result := assertions.So(actual, so, expected...)
	self.Log("    + ", description+"\n")
	if !ok {
		self.t.Fail()
		self.Log(self.formatResult(description, result))
	}
}

func (self *Fixture) SkipSo(description string, actual interface{}, so func(actual interface{}, expected ...interface{}) string, expected ...interface{}) {
	self.Log("    + (skipped) ", description+"\n")
}

func (self *Fixture) formatResult(description, result string) string {
	_, file, line, _ := runtime.Caller(2)
	fileInfo := file + ":" + strconv.Itoa(line)
	title := "FAILED: \"" + description + "\""
	divider := strings.Repeat("*", max(len(fileInfo), len(title)))
	message := "\n    " + divider + "\n\n    " + title + "\n\n"
	for _, line := range strings.Split(result, "\n") {
		message += "    " + line + "\n"
	}
	return message + "\n\n    " + fileInfo + "\n\n    " + divider + "\n\n"
}

func (self *Fixture) Log(args ...interface{}) {
	self.output.WriteString(fmt.Sprint(args...))
}

func (self *Fixture) Logf(message string, args ...interface{}) {
	self.output.WriteString(fmt.Sprintf(message, args...))
}

// A represents an abbreviation of the function signatures implemented by the
// functions in `github.com/smartystreets/goconvey/convey/assertions`.
type A func(
	description string,
	actual interface{},
	so func(actual interface{}, expected ...interface{}) string,
	expected ...interface{},
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

//////////////////////////////////////////////////////////////////////////////

var (
	So                   = assertions.So
	ShouldEqual          = assertions.ShouldEqual
	ShouldNotEqual       = assertions.ShouldNotEqual
	ShouldAlmostEqual    = assertions.ShouldAlmostEqual
	ShouldNotAlmostEqual = assertions.ShouldNotAlmostEqual
	ShouldResemble       = assertions.ShouldResemble
	ShouldNotResemble    = assertions.ShouldNotResemble
	ShouldPointTo        = assertions.ShouldPointTo
	ShouldNotPointTo     = assertions.ShouldNotPointTo
	ShouldBeNil          = assertions.ShouldBeNil
	ShouldNotBeNil       = assertions.ShouldNotBeNil
	ShouldBeTrue         = assertions.ShouldBeTrue
	ShouldBeFalse        = assertions.ShouldBeFalse
	ShouldBeZeroValue    = assertions.ShouldBeZeroValue

	ShouldBeGreaterThan          = assertions.ShouldBeGreaterThan
	ShouldBeGreaterThanOrEqualTo = assertions.ShouldBeGreaterThanOrEqualTo
	ShouldBeLessThan             = assertions.ShouldBeLessThan
	ShouldBeLessThanOrEqualTo    = assertions.ShouldBeLessThanOrEqualTo
	ShouldBeBetween              = assertions.ShouldBeBetween
	ShouldNotBeBetween           = assertions.ShouldNotBeBetween
	ShouldBeBetweenOrEqual       = assertions.ShouldBeBetweenOrEqual
	ShouldNotBeBetweenOrEqual    = assertions.ShouldNotBeBetweenOrEqual

	ShouldContain    = assertions.ShouldContain
	ShouldNotContain = assertions.ShouldNotContain
	ShouldBeIn       = assertions.ShouldBeIn
	ShouldNotBeIn    = assertions.ShouldNotBeIn
	ShouldBeEmpty    = assertions.ShouldBeEmpty
	ShouldNotBeEmpty = assertions.ShouldNotBeEmpty

	ShouldStartWith           = assertions.ShouldStartWith
	ShouldNotStartWith        = assertions.ShouldNotStartWith
	ShouldEndWith             = assertions.ShouldEndWith
	ShouldNotEndWith          = assertions.ShouldNotEndWith
	ShouldBeBlank             = assertions.ShouldBeBlank
	ShouldNotBeBlank          = assertions.ShouldNotBeBlank
	ShouldContainSubstring    = assertions.ShouldContainSubstring
	ShouldNotContainSubstring = assertions.ShouldNotContainSubstring

	ShouldPanic        = assertions.ShouldPanic
	ShouldNotPanic     = assertions.ShouldNotPanic
	ShouldPanicWith    = assertions.ShouldPanicWith
	ShouldNotPanicWith = assertions.ShouldNotPanicWith

	ShouldHaveSameTypeAs    = assertions.ShouldHaveSameTypeAs
	ShouldNotHaveSameTypeAs = assertions.ShouldNotHaveSameTypeAs
	ShouldImplement         = assertions.ShouldImplement
	ShouldNotImplement      = assertions.ShouldNotImplement

	ShouldHappenBefore         = assertions.ShouldHappenBefore
	ShouldHappenOnOrBefore     = assertions.ShouldHappenOnOrBefore
	ShouldHappenAfter          = assertions.ShouldHappenAfter
	ShouldHappenOnOrAfter      = assertions.ShouldHappenOnOrAfter
	ShouldHappenBetween        = assertions.ShouldHappenBetween
	ShouldHappenOnOrBetween    = assertions.ShouldHappenOnOrBetween
	ShouldNotHappenOnOrBetween = assertions.ShouldNotHappenOnOrBetween
	ShouldHappenWithin         = assertions.ShouldHappenWithin
	ShouldNotHappenWithin      = assertions.ShouldNotHappenWithin
	ShouldBeChronological      = assertions.ShouldBeChronological
)
