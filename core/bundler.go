package core

import (
  "fmt"
)

// The Bundler has two distinct tasks:
// - Accept local events, bundle them when the frame advance, and send them
// to the Updater.
// - Accept time deltas from the Auditor and speed up or slow down the ticker
// as necessary to stay in sync with other engines.  Since the Auditor is
// responsible for making sure all data that has been received is valid it
// will signal to the Bundler to pause if it needs to wait for other engines
// to catch up.
type Bundler struct {
  Params EngineParams
  Ticker Ticker

  // Used to receive events generated locally.
  Local_event        <-chan Event
  Local_engine_event <-chan EngineEvent

  // If this engine is out of sync with other engines the Auditor will tell us
  // how much to adjust our clock by via this channel.
  Time_delta <-chan int64

  // Bundles of events generated locally.  These are packaged up and sent to
  // the updater when they are ready.
  Local_bundles chan<- FrameBundle

  Current_ms int64

  shutdown chan struct{}
}

func (b *Bundler) Start() {
  b.shutdown = make(chan struct{})
  go b.routine()
}

func (b *Bundler) routine() {
  b.Ticker.Start()
  current_frame := StateFrame(b.Current_ms / b.Params.Frame_ms)
  var current_events []Event
  var current_engine_events []EngineEvent
  for {
    select {
    case <-b.shutdown:
      // TODO: Drain channels and free stuff up?
      close(b.Local_bundles)
      return

    case event := <-b.Local_event:
      current_events = append(current_events, event)

    case engine_event := <-b.Local_engine_event:
      // TODO: We can send the frame the engine event will be applied on,
      // is this important
      // b.engine_event_frame <- current_frame
      current_engine_events = append(current_engine_events, engine_event)

    case <-b.Ticker.Chan():
      b.Current_ms++
      next_frame := StateFrame(b.Current_ms / b.Params.Frame_ms)
      for ; current_frame < next_frame; current_frame++ {
        if b.Params.Id != 1234 {
          fmt.Printf("Send_local(%d): %d\n", b.Params.Id%1000, current_frame)
        }
        b.Local_bundles <- FrameBundle{
          Frame: current_frame,
          Bundle: EventBundle{
            b.Params.Id: AllEvents{
              Game:   current_events,
              Engine: current_engine_events,
            },
          },
        }
        current_events = nil
        current_engine_events = nil
      }

    case delta := <-b.Time_delta:
      b.Current_ms += delta
    }
  }
}

func (b *Bundler) Shutdown() {
  b.shutdown <- struct{}{}
}
