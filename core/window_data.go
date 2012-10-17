package core

import (
  "fmt"
)

type DataWindow struct {
  first StateFrame // lowest valued p that can access the window
  start int        // index of p's position in the window
  data  []FrameData
}

func NewDataWindow(n int, start StateFrame) *DataWindow {
  return &DataWindow{
    first: start,
    start: 0,
    data:  make([]FrameData, n),
  }
}

func (w *DataWindow) posToIndex(pos StateFrame) int {
  index := (w.start + int(pos-w.first)) % len(w.data)
  if pos < w.first || pos >= w.first+StateFrame(len(w.data)) {
    panic(fmt.Sprintf("Tried to access %d, which is outside of window bounds, %d - %d.", pos, w.first, int(w.first)+len(w.data)))
  }
  return index
}

func (w *DataWindow) Get(pos StateFrame) FrameData {
  index := w.posToIndex(pos)
  return w.data[index]
}

func (w *DataWindow) Set(pos StateFrame, val FrameData) {
  index := w.posToIndex(pos)
  w.data[index] = val
}

func (w *DataWindow) Start() StateFrame {
  return w.first
}

func (w *DataWindow) End() StateFrame {
  return w.first + StateFrame(len(w.data))
}

func (w *DataWindow) Advance() {
  w.first++
  w.start = (w.start + 1) % (len(w.data))
}
