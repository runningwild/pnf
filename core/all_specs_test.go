package core_test

import (
  "encoding/gob"
  "runningwild/pnf/core"
  "github.com/orfjackal/gospec/src/gospec"
  "testing"
)

type EventA struct {
  Data int
}

func init() {
  gob.Register(EventA{})
}
func (e EventA) ApplyFirst(core.Game) {}
func (e EventA) Apply(g core.Game) {
  g.(*TestGame).A += e.Data
}
func (e EventA) ApplyFinal(core.Game) {}

type EventB struct {
  Data string
}

func init() {
  gob.Register(EventA{})
}
func (e EventB) ApplyFirst(core.Game) {}
func (e EventB) Apply(g core.Game) {
  g.(*TestGame).B = e.Data
}
func (e EventB) ApplyFinal(core.Game) {}

type TestGame struct {
  A      int
  B      string
  Thinks int
}

func (g *TestGame) ThinkFirst()  {}
func (g *TestGame) ThinkFinal() {}
func (g *TestGame) Think() {
  g.Thinks++
}
func (g *TestGame) Copy() core.Game {
  g2 := *g
  println("Original: ", g.Thinks)
  println("Copy: ", g2.Thinks)
  return &g2
}

func TestAllSpecs(t *testing.T) {
  r := gospec.NewRunner()
  r.AddSpec(NetworkMockSpec)
  r.AddSpec(BundlerSpec)
  r.AddSpec(UpdaterSpec)
  gospec.MainGoTest(r, t)
}
