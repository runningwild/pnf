package pnf

// FastX happens once, potentially before any events arrive
// X can happen multiple times, whenever anything changes
// FinalX only happens after all events have arrived

type Event interface {
  // Cannot modify the Game
  ApplyFast(Game)

  // Can modify the game
  Apply(Game)
  ApplyFinal(Game)
}

type Game interface {
  ThinkFast()
  Think()
  ThinkFinal()
}

// Engine needs to be able to host, join, and communicate events
// There should be a network manager interface that it uses

type player struct {
  x, y int
}
type myGame struct {
  players []player
}
