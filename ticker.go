package pnf

type Ticker interface {
  Start(ms int)
  Stop()
  Chan() <-chan int
}

type FakeTicker struct {
  cur   int
  delta int
  c     chan int
}

func NewFakeTicker() *FakeTicker {
  return &FakeTicker{}
}

func (f *FakeTicker) Start(delta int) {
  if f.delta != 0 {
    panic("Started an already started FakeTicker.")
  }
  if delta <= 0 {
    panic("Cannot start FakeTicker with delta <= 0.")
  }
  f.delta = delta
  f.c = make(chan int)
}

func (f *FakeTicker) Stop() {
  if f.delta <= 0 {
    panic("Cannot stop a FakeTicker that has not been started yet.")
  }
  f.delta = 0
  close(f.c)
}

func (f *FakeTicker) Chan() <-chan int {
  return f.c
}

func (f *FakeTicker) Inc(ms int) {
  for i := 0; i < ms; i++ {
    f.cur++
    if f.cur%f.delta == 0 {
      f.c <- f.cur
    }
  }
}
