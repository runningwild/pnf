package core

type Ticker interface {
  Start()
  Stop()
  Chan() <-chan struct{}
}

type FakeTicker struct {
  c chan struct{}
}

func (f *FakeTicker) Start() {
  if f.c != nil {
    panic("Started an already started FakeTicker.")
  }
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
