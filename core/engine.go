package core

type EngineId int64
type StateFrame int

type EngineParams struct {
  Id         int64
  Delay_ms   int64
  Frame_ms   int64
  Max_frames int
}
type Engine struct {
  network Network

  params EngineParams
  states GameWindow
  events EventsWindow
  ticker Ticker

  local_events      []Event
  local_events_chan chan Event
}
