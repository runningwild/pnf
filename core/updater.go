package core

// Used for bootstrapping
type BootstrapFrame struct {
  Frame StateFrame
  Game  Game
  Info  EngineInfo
}

// The updater has the following tasks:
// Receive Events from all engines, including localhost, store the events and
// apply them to the Game as necessary.  If events show up late it will rewind
// the Game to an older state and reapply all Events in the proper order.
type Updater struct {
  Params EngineParams

  // FrameBundles formed by this engine.  These will always come in order and
  // have their StateFrame attached, so this channel also serves as a clock.
  Local_bundles <-chan FrameBundle

  // Local bundles are sent through here to the Communicator to be broadcast
  // to other engines.
  Broadcast_bundles chan<- FrameBundle

  // Bundles formed by remote engines.  These come from the Auditor and can
  // come in any order so it is the updater's responsibility to make sure that
  // they are applied properly.
  Remote_bundles <-chan FrameBundle

  // Whenever a frame is completed it is sent to the Communicator through here
  // to be used for bootstrapping new connections.  An engine that does not
  // want to host can safely leave this as nil.
  Bootstrap_frames chan<- BootstrapFrame

  // If this is non-negative then local events occurring on a frame before
  // this are ignored.  This is for bootstrapping engines that haven't figured
  // out what frame they should join the game on yet.  While an engine is
  // bootstrapping this will be set to -1 to indicate that local bundles
  // should be discarded for now.
  skip_to_frame StateFrame

  // Requests for game states are made along this channel and a response is
  // given immediately, or stored in final_requests or fast_requests to be
  // fulfilled later.
  request_state  chan stateRequest
  final_requests []stateRequest
  fast_requests  []stateRequest

  // Requests for certain information can be made along this channel.
  // Currently the only use is to check the number of connected players.
  info_request  chan struct{}
  info_response chan int

  // These windows store the Game states and EventBundles for each StateFrame.
  // The windows will advance as soon as all events for a given frame have
  // been received.
  data_window *DataWindow

  // The most recent frame for which we've received a FrameBundle for
  // localhost.
  local_frame StateFrame

  // The most recent frame for which we've received a FrameBundle from any
  // engine.
  global_frame StateFrame

  // The oldest frame for which we've received new Events but haven't gone
  // back yet to reThink.
  oldest_dirty_frame StateFrame

  // Shuts everything down and closes all channels it sends on.
  shutdown chan struct{}
}

func (u *Updater) Start(frame StateFrame, data FrameData) {
  u.data_window = NewDataWindow(u.Params.Max_frames+1, frame)
  u.data_window.Set(frame, data)
  for i := u.data_window.Start(); i < u.data_window.End(); i++ {
    future_data := u.data_window.Get(i)
    future_data.Game = data.Game.Copy().(Game)
    u.data_window.Set(i, future_data)
  }
  u.local_frame = frame
  u.global_frame = frame
  u.oldest_dirty_frame = frame + 1
  u.request_state = make(chan stateRequest)
  u.info_request = make(chan struct{})
  u.info_response = make(chan int)
  go u.routine()
}

type stateRequest struct {
  frame    StateFrame
  response chan stateResponse
  final    bool
}
type stateResponse struct {
  game  Game
  frame StateFrame
}

func (u *Updater) Bootstrap(boot *BootstrapFrame) {
  u.data_window = NewDataWindow(u.Params.Max_frames+1, boot.Frame)
  dummy_bundles := make(EventBundle)
  for engine_id := range boot.Info.Engines {
    dummy_bundles[engine_id] = AllEvents{}
  }
  for i := u.data_window.Start(); i < u.data_window.End(); i++ {
    future_data := u.data_window.Get(i)
    future_data.Game = boot.Game.Copy().(Game)
    u.data_window.Set(i, future_data)
  }
  u.data_window.Set(boot.Frame, FrameData{
    Bundle: dummy_bundles, // So we can advance past it any time
    Game:   boot.Game.Copy().(Game),
    Info:   boot.Info,
  })
  u.data_window.Set(boot.Frame+1, FrameData{
    Bundle: make(EventBundle),
    Game:   boot.Game.Copy().(Game), // Really just a placeholder
    Info:   boot.Info,               // Prevents us from proceeding too early
  })
  u.skip_to_frame = -1
  u.local_frame = boot.Frame + 1
  u.global_frame = boot.Frame + 1
  u.oldest_dirty_frame = boot.Frame + 2
  u.request_state = make(chan stateRequest)
  u.info_request = make(chan struct{})
  u.info_response = make(chan int)
  go u.routine()
}

// Go through any pending final state requests for the game state and
// fulfill any that are ready.
func (u *Updater) fulfillFinalRequests() {
  for i := 0; i < len(u.final_requests); i++ {
    if u.final_requests[i].frame == u.data_window.Start() {
      u.final_requests[i].response <- stateResponse{
        game:  u.data_window.Get(u.data_window.Start()).Game,
        frame: u.data_window.Start(),
      }
      u.final_requests[i] = u.final_requests[len(u.final_requests)-1]
      u.final_requests = u.final_requests[0 : len(u.final_requests)-1]
    }
  }
}

// Go through any pending fast state requests for the game state and
// fulfill any that are ready.
func (u *Updater) fulfillFastRequests() {
  for i := 0; i < len(u.fast_requests); i++ {
    if u.fast_requests[i].frame == u.local_frame {
      u.fast_requests[i].response <- stateResponse{
        game:  u.data_window.Get(u.local_frame).Game,
        frame: u.local_frame,
      }
      u.fast_requests[i] = u.fast_requests[len(u.fast_requests)-1]
      u.fast_requests = u.fast_requests[0 : len(u.fast_requests)-1]
    }
  }
}

// Does a rethink on every dirty frame and then advances data_window as much
// as possible.
func (u *Updater) advance() {
  prev_data := u.data_window.Get(u.oldest_dirty_frame - 1)
  for frame := u.oldest_dirty_frame; frame <= u.global_frame; frame++ {
    data := u.data_window.Get(frame)
    data.Game.OverwriteWith(prev_data.Game)
    new_info := prev_data.Info.Copy()
    data.Info = new_info
    data.Bundle.EachEngine(frame, func(id EngineId, events []EngineEvent) {
      for _, event := range events {
        event.Apply(&data.Info)
      }
    })
    data.Bundle.Each(frame, func(id EngineId, events []Event) {
      if _, ok := data.Info.Engines[id]; !ok {
        // TODO: What on earth to do about this?
        return
      }
      for _, event := range events {
        event.Apply(data.Game)
      }
    })

    // A nil set of Engines is the signal that this is a bootstrap game state,
    // so we should not think on it and just copy it to the next frame.
    if data.Info.Engines != nil {
      data.Game.Think()
    }
    u.data_window.Set(frame, data)
    prev_data = data
  }
  u.oldest_dirty_frame = u.global_frame + 1

  // As long as the *second* frame in the window is complete we can advance,
  // this way we always keep around one frame to copy from if we need it.
  for u.data_window.Start() < u.global_frame {
    prev_info := u.data_window.Get(u.data_window.Start() + 1).Info
    data := u.data_window.Get(u.data_window.Start() + 1)
    all_present := true
    for id := range prev_info.Engines {
      if _, ok := data.Bundle[id]; !ok {
        all_present = false
      }
    }
    if all_present {
      u.data_window.Advance()
      if u.Bootstrap_frames != nil {
        bootstrap_frame := BootstrapFrame{
          Frame: u.data_window.Start(),
          Game:  data.Game,
          Info:  data.Info,
        }
        u.Bootstrap_frames <- bootstrap_frame
      }
      // As soon as we get to a final state we check to see if anyone was
      // waiting on it.
      u.fulfillFinalRequests()
    } else {
      break
    }
  }
}

func (u *Updater) initFrameData(frame StateFrame) {
  if frame-1 < u.data_window.Start() {
    return
  }
  prev_data := u.data_window.Get(frame - 1)
  data := u.data_window.Get(frame)
  data.Bundle = make(EventBundle)
  data.Info = prev_data.Info.Copy()
  u.data_window.Set(frame, data)
}

func (u *Updater) routine() {
  for {
    select {
    case local_bundle := <-u.Local_bundles:
      if u.skip_to_frame == -1 || local_bundle.Frame < u.skip_to_frame {
        continue
      }
      if u.skip_to_frame > 0 {
        if u.skip_to_frame < u.oldest_dirty_frame {
          u.oldest_dirty_frame = u.skip_to_frame
        }
        for frame := u.skip_to_frame; frame < local_bundle.Frame; frame++ {
          data := u.data_window.Get(frame)
          dummy_bundle := EventBundle(map[EngineId]AllEvents{u.Params.Id: AllEvents{}})
          data.Bundle.AbsorbEventBundle(dummy_bundle)
          u.Broadcast_bundles <- FrameBundle{
            Bundle: dummy_bundle,
            Frame:  frame,
          }
          u.data_window.Set(frame, data)
        }
        u.skip_to_frame = 0
      }
      u.local_frame = local_bundle.Frame
      start := u.global_frame + 1
      if start < u.skip_to_frame {
        start = u.skip_to_frame
      }
      // TODO: Check that the local bundle is in bounds
      for frame := start; frame <= u.local_frame; frame++ {
        u.initFrameData(frame)
      }
      if u.global_frame < u.local_frame {
        u.global_frame = u.local_frame
      }
      if u.local_frame < u.oldest_dirty_frame {
        u.oldest_dirty_frame = u.local_frame
      }
      data := u.data_window.Get(local_bundle.Frame)
      data.Bundle.AbsorbEventBundle(local_bundle.Bundle)
      u.data_window.Set(local_bundle.Frame, data)
      u.Broadcast_bundles <- local_bundle
      u.advance()
      u.fulfillFastRequests()

    case remote_bundle := <-u.Remote_bundles:
      // When bootstrapping it is totally possible to get events before our
      // world begins, so we need to make sure to discard those.
      if remote_bundle.Frame <= u.data_window.Start() {
        continue
      }
      for frame := u.global_frame + 1; frame <= remote_bundle.Frame; frame++ {
        u.initFrameData(frame)
      }
      if u.global_frame < remote_bundle.Frame {
        u.global_frame = remote_bundle.Frame
      }
      if remote_bundle.Frame < u.oldest_dirty_frame {
        u.oldest_dirty_frame = remote_bundle.Frame
      }
      if u.skip_to_frame == -1 {
        remote_bundle.Bundle.EachEngine(remote_bundle.Frame, func(id EngineId, events []EngineEvent) {
          for _, event := range events {
            if joined, ok := event.(EngineJoined); ok && joined.Id == u.Params.Id {
              u.skip_to_frame = remote_bundle.Frame
            }
          }
        })
      }
      // TODO: Check that the remote bundle is in bounds
      data := u.data_window.Get(remote_bundle.Frame)
      data.Bundle.AbsorbEventBundle(remote_bundle.Bundle)
      u.data_window.Set(remote_bundle.Frame, data)
      u.advance()

    case req := <-u.request_state:
      if req.final {
        switch {
        case req.frame < 0 || req.frame == u.data_window.Start():
          req.response <- stateResponse{
            game:  u.data_window.Get(u.data_window.Start()).Game,
            frame: u.data_window.Start(),
          }
        case req.frame < u.data_window.Start():
          req.response <- stateResponse{}
        default:
          u.final_requests = append(u.final_requests, req)
        }
      } else {
        switch {
        case req.frame < 0:
          req.response <- stateResponse{
            game:  u.data_window.Get(u.local_frame).Game,
            frame: u.local_frame,
          }
        case req.frame < u.data_window.Start():
          req.response <- stateResponse{}
        case req.frame <= u.local_frame:
          req.response <- stateResponse{
            game:  u.data_window.Get(req.frame).Game,
            frame: req.frame,
          }
        default:
          u.fast_requests = append(u.fast_requests, req)
        }
      }

    case <-u.info_request:
      info := u.data_window.Get(u.data_window.Start()).Info
      u.info_response <- len(info.Engines)

    case <-u.shutdown:
      close(u.Broadcast_bundles)
      return
    }
  }
}

// Pass frame < 0 to get the most recent final frame
func (u *Updater) RequestFinalGameState(frame StateFrame) (Game, StateFrame) {
  response := make(chan stateResponse)
  u.request_state <- stateRequest{frame, response, true}
  data := <-response
  return data.game, data.frame
}

func (u *Updater) RequestFastGameState(frame StateFrame) (Game, StateFrame) {
  response := make(chan stateResponse)
  u.request_state <- stateRequest{frame, response, false}
  data := <-response
  return data.game, data.frame
}

func (u *Updater) NumEngines() int {
  u.info_request <- struct{}{}
  return <-u.info_response
}

func (u *Updater) Shutdown() {
  u.shutdown <- struct{}{}
}
