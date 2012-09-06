package core

import (
  "time"
)

type Ticker interface {
  Start()
  Stop()
  Chan() <-chan struct{}
}

type BasicTicker struct {
  ticker *time.Ticker
  c      chan struct{}
}

func NewBasicTicker() Ticker {
  return &BasicTicker{}
}
func (bt *BasicTicker) Start() {
  if bt.ticker != nil {
    panic("Started an already started BasicTicker.")
  }
  bt.ticker = time.NewTicker(time.Millisecond)
  bt.c = make(chan struct{})
  go func() {
    for _ = range bt.ticker.C {
      bt.c <- struct{}{}
    }
  }()
}
func (bt *BasicTicker) Stop() {
  if bt.ticker == nil {
    panic("Cannot stop a BasicTicker that has not been started yet.")
  }
  bt.ticker.Stop()
  bt.ticker = nil
}
func (bt *BasicTicker) Chan() <-chan struct{} {
  return bt.c
}

type FakeTicker struct {
  c chan struct{}
}

func (f *FakeTicker) Start() {
  if f.c != nil {
    panic("Started an already started FakeTicker.")
  }
  println("Fake ticker started")
  f.c = make(chan struct{})
}

func (f *FakeTicker) Stop() {
  if f.c == nil {
    panic("Cannot stop a FakeTicker that has not been started yet.")
  }
  close(f.c)
  f.c = nil
}

func (f *FakeTicker) Chan() <-chan struct{} {
  return f.c
}

func (f *FakeTicker) Inc(ms int) {
  for i := 0; i < ms; i++ {
    f.c <- struct{}{}
  }
}
