package gounit

import (
	"testing"

	. "github.com/mdwhatcott/gounit"
)

func TestScoring(t *testing.T) {
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

///////////////////////////////////////////////////////

type Game struct {
	score int
}

func NewGame() *Game {
	return &Game{}
}

func (self *Game) Roll(pins int) {
	self.score += pins
}

func (self *Game) Score() int {
	return self.score
}

func (self *Game) rollMany(pins, times int) {
	for x := 0; x < times; x++ {
		self.Roll(pins)
	}
}
