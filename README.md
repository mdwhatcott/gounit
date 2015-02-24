# gounit

Package gounit implements xunit for Go (along with some other goodies).

http://en.wikipedia.org/wiki/XUnit

(No attempt has yet been made to produce XUnit-style XML output.)

## Installation:

    go get -u "github.com/mdwhatcott/gounit"

That command will take some time because gounit depends on `github.com/smartystreets/goconvey/convey/assertions`, a really handy set of assertions. That assertions package is part of the [goconvey](http://goconvey.co) testing project, which has a lot more going on and is somewhat large. Have no fear though, because you only need import the package in your `*_test.go` files your finished binaries will have no trace of any of these testing dependencies.

## Recommended Import:

    import . "github.com/mdwhatcott/gounit"

Notice the '.' which brings all exported gounit names into your testing environment. This removes lots of clutter from your test and will allow you to see the important parts of your tests more easily.

## Complete Example Test Function:

```go
package examples

import (
    "testing"

    . "github.com/mdwhatcott/gounit"
)

func Test(t *testing.T) {
    var game *Game

    f := NewFixture("Bowling Game Score", t)
    defer f.Run()

    f.Setup(func() {
        game = NewGame()
    })
    f.Test("After rolling all gutter balls", func() {
        game.rollMany(0, 20)
        f.So("No points will be earned", game.Score(), ShouldEqual, 0)
    })
    f.Test("All balls knock down a single pin--score of 20", func() {
        game.rollMany(1, 20)
        f.So("Each roll will score a point", game.Score(), ShouldEqual, 20)
    })
    f.SkipTest("Spare earns bonus", func() {
        game.Roll(3)
        game.Roll(7)
        game.Roll(3)
        game.rollMany(0, 17)
        f.So("The roll after the spare should be counted twice",
            game.Score(), ShouldEqual, 16)
    })
}

```

## Corresponding Output:

```
$ go test -v
=== RUN Test
--- FAIL: Test (0.00s)
    gounit.go:208: Bowling Game Score
         -> "After rolling all gutter balls"
            + No points will be earned
        
            ************************************************************************
        
            FAILED: "No points will be earned"
        
            Expected: '0'
            Actual:   '1'
            (Should be equal)
        
        
            /Users/mike/src/github.com/mdwhatcott/gounit/examples/bowling_test.go:20
        
            ************************************************************************
        
         -> "All balls knock down a single pin--score of 20"
            + Each roll will score a point
         -> (skipped) "Spare earns bonus"

FAIL
exit status 1
FAIL    github.com/mdwhatcott/gounit/examples   0.007s
```

## Documentation:

[godoc.org/github.com/mdwhatcott/gounit](http://godoc.org/github.com/mdwhatcott/gounit)