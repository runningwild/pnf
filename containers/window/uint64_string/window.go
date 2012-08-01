package window


import (
  "fmt"
)

type Window struct {
  first uint64   // lowest valued p that can access the window
  start int // index of p's position in the window
  data  []string
}

func New(n int, start uint64) *Window {
  return &Window{
    first: start,
    start: 0,
    data:  make([]string, n),
  }
}

func (w *Window) posToIndex(pos uint64) int {
  index := (w.start + int(pos - w.first)) % len(w.data)
  if index < 0 || index > len(w.data) {
    panic(fmt.Sprintf("Tried to access %d, which is outside of window bounds, %d - %d.", pos, w.first, int(w.first)+len(w.data)))
  }
  return index
}

func (w *Window) Get(pos uint64) string {
  index := w.posToIndex(pos)
  return w.data[index]
}

func (w *Window) Set(pos uint64, val string) {
  index := w.posToIndex(pos)
  w.data[index] = val
}

func (w *Window) Start() uint64 {
  return w.first
}

func (w *Window) End() uint64 {
  return w.first + uint64(len(w.data))
}

func (w *Window) Advance() {
  w.first++
  var V string
  w.data[w.start] = V
  w.start = (w.start + 1) % (len(w.data))
}
// Here we will test that the types parameters are ok...
func testTypes(arg0 uint64, arg1 string) {
    f := func(interface{}, interface{}) { } // this func does nothing...
    f(arg0, arg1)
}
