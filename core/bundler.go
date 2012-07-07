package core

// The Bundler has two distinct tasks:
// - Accept local events, bundle them when the frame advance, and send them
// to the Updater.
// - Accept time deltas from the Auditor and speed up or slow down the ticker
// as necessary to stay in sync with other engines.  Since the Auditor is
// responsible for making sure all data that has been received is valid it
// will signal to the Bundler to pause if it needs to wait for other engines
// to catch up.
type Bundler struct {
  Params          EngineParams
  Ticker          Ticker
  Local_event     <-chan Event
  Time_delta      <-chan int64
  Completed_frame <-chan StateFrame
  Local_bundles   chan<- FrameBundle
  Current_ms      int64

  shutdown             chan struct{}
  current_event_bundle []Event
}

func (b *Bundler) Start() {
  b.shutdown = make(chan struct{})
  go b.routine()
}

func (b *Bundler) routine() {
  b.Ticker.Start()
  current_frame := StateFrame(b.Current_ms / b.Params.Frame_ms)
  completed := current_frame - 1
  for {
    select {
    case <-b.shutdown:
      // TODO: Drain channels and free stuff up?
      close(b.Local_bundles)
      return

    case event := <-b.Local_event:
      b.current_event_bundle = append(b.current_event_bundle, event)

    case <-b.Ticker.Chan():
      b.Current_ms++
      next_frame := StateFrame(b.Current_ms / b.Params.Frame_ms)
      for ; current_frame < next_frame; current_frame++ {
        b.Local_bundles <- FrameBundle{
          Frame: current_frame,
          Bundle: EventBundle{
            b.Params.Id: b.current_event_bundle,
          },
        }
        b.current_event_bundle = nil
      }

    case delta := <-b.Time_delta:
      b.Current_ms += delta

    case newest_completed := <-b.Completed_frame:
      if newest_completed > completed {
        completed = newest_completed
      }
    }
  }
}

func (b *Bundler) Shutdown() {
  b.shutdown <- struct{}{}
}
