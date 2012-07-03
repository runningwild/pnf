package core

import (
  "fmt"
)

type GameWindow struct {
  first StateFrame // lowest valued p that can access the Gamewindow
  start int        // index of p's position in the Gamewindow
  data  []Game
}

func NewGameWindow(n int, start StateFrame) *GameWindow {
  return &GameWindow{
    first: start,
    start: 0,
    data:  make([]Game, n),
  }
}

func (w *GameWindow) posToIndex(pos StateFrame) int {
  index := (w.start + int(pos-w.first)) % len(w.data)
  if index < 0 || index > len(w.data) {
    panic(fmt.Sprintf("Tried to access %d, which is outside of Gamewindow bounds, %d - %d.", pos, w.first, int(w.first)+len(w.data)))
  }
  return index
}

func (w *GameWindow) Get(pos StateFrame) Game {
  index := w.posToIndex(pos)
  return w.data[index]
}

func (w *GameWindow) Set(pos StateFrame, val Game) {
  index := w.posToIndex(pos)
  w.data[index] = val
}

func (w *GameWindow) Start() StateFrame {
  return w.first
}

func (w *GameWindow) End() StateFrame {
  return w.first + StateFrame(len(w.data))
}

func (w *GameWindow) Advance() {
  w.first++
  w.start = (w.start + 1) % (len(w.data))
}
