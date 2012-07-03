package pnf

import (
  "runningwild/pnf/core"
)

type Game interface {
  core.Game
}

type Engine struct {
  engine core.Engine
}

type RemoteHost struct{}

func (e *Engine) Host(ping, join func([]byte) []byte) {}
func (e *Engine) FindHosts(data []byte) []RemoteHost {
  return nil
}
func (e *Engine) JoinHost(data []byte) {}

func (e *Engine) Start(game Game) {}

func NewEngine(params string) *Engine {
  // var n core.Network
  return nil
}
