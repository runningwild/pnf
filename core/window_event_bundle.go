package core

import (
  "fmt"
)

type EventBundleWindow struct {
  first StateFrame // lowest valued p that can access the window
  start int        // index of p's position in the window
  data  []EventBundle
}

func NewEventBundleWindow(n int, start StateFrame) *EventBundleWindow {
  return &EventBundleWindow{
    first: start,
    start: 0,
    data:  make([]EventBundle, n),
  }
}

func (w *EventBundleWindow) posToIndex(pos StateFrame) int {
  index := (w.start + int(pos-w.first)) % len(w.data)
  if index < 0 || index > len(w.data) {
    panic(fmt.Sprintf("Tried to access %d, which is outside of window bounds, %d - %d.", pos, w.first, int(w.first)+len(w.data)))
  }
  return index
}

func (w *EventBundleWindow) Get(pos StateFrame) EventBundle {
  index := w.posToIndex(pos)
  return w.data[index]
}

func (w *EventBundleWindow) Set(pos StateFrame, val EventBundle) {
  index := w.posToIndex(pos)
  w.data[index] = val
}

func (w *EventBundleWindow) Start() StateFrame {
  return w.first
}

func (w *EventBundleWindow) End() StateFrame {
  return w.first + StateFrame(len(w.data))
}

func (w *EventBundleWindow) Advance() {
  w.first++
  w.start = (w.start + 1) % (len(w.data))
}

// Here we will test that the types parameters are ok...
func testTypes(arg0 StateFrame, arg1 EventBundle) {
  f := func(interface{}, interface{}) {} // this func does nothing...
  f(arg0, arg1)
}
