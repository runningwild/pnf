package pnf

import (
  "fmt"
)

type EventsWindow struct {
  first StateFrame // lowest valued p that can access the Eventswindow
  start int        // index of p's position in the Eventswindow
  data  [][]Event
}

func NewEventsWindow(n int, start StateFrame) *EventsWindow {
  return &EventsWindow{
    first: start,
    start: 0,
    data:  make([][]Event, n),
  }
}

func (w *EventsWindow) posToIndex(pos StateFrame) int {
  index := (w.start + int(pos-w.first)) % len(w.data)
  if index < 0 || index > len(w.data) {
    panic(fmt.Sprintf("Tried to access %d, which is outside of Eventswindow bounds, %d - %d.", pos, w.first, int(w.first)+len(w.data)))
  }
  return index
}

func (w *EventsWindow) Get(pos StateFrame) []Event {
  index := w.posToIndex(pos)
  return w.data[index]
}

func (w *EventsWindow) Set(pos StateFrame, val []Event) {
  index := w.posToIndex(pos)
  w.data[index] = val
}

func (w *EventsWindow) Start() StateFrame {
  return w.first
}

func (w *EventsWindow) End() StateFrame {
  return w.first + StateFrame(len(w.data))
}

func (w *EventsWindow) Advance() {
  w.first++
  w.start = (w.start + 1) % (len(w.data))
}
