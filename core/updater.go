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

  // Used to signal that the current Game state has been requested.  True
  // indicates that we should send the most recent complete Game state, false
  // indicates that we should send the most recent Game state, even if
  // incomplete.
  request_state chan bool

  // When requested the current Game state is sent through here.
  current_state chan Game

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
  u.local_frame = frame
  u.global_frame = frame
  u.oldest_dirty_frame = frame + 1
  u.request_state = make(chan bool)
  u.current_state = make(chan Game)
  go u.routine()
}

func (u *Updater) Bootstrap(boot *BootstrapFrame) {
  u.data_window = NewDataWindow(u.Params.Max_frames+1, boot.Frame-1)
  u.data_window.Set(boot.Frame-1, FrameData{
    Bundle: make(EventBundle),
    Game:   boot.Game,
    Info:   EngineInfo{}, // This will let us advance past it
  })
  u.data_window.Set(boot.Frame, FrameData{
    Bundle: make(EventBundle),
    Game:   boot.Game,
    Info:   boot.Info,
  })
  u.local_frame = boot.Frame
  u.global_frame = boot.Frame
  u.oldest_dirty_frame = boot.Frame + 1
  println("Setting oldest dirty frame to ", u.oldest_dirty_frame)
  u.request_state = make(chan bool)
  u.current_state = make(chan Game)
  go u.routine()
}

// Does a rethink on every dirty frame and then advances data_window as much
// as possible.
func (u *Updater) advance() {
  prev_data := u.data_window.Get(u.oldest_dirty_frame - 1)
  for frame := u.oldest_dirty_frame; frame <= u.global_frame; frame++ {
    data := u.data_window.Get(frame)
    data.Game = prev_data.Game.Copy().(Game)
    data.Bundle.EachEngine(frame, func(id EngineId, events []EngineEvent) {
      for _, event := range events {
        event.Apply(&data.Info)
        println("Applied engine event on frame ", frame)
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
    data.Game.Think()
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
  var data FrameData
  data.Game = prev_data.Game.Copy().(Game)
  data.Bundle = make(EventBundle)
  data.Info = prev_data.Info.Copy()
  u.data_window.Set(frame, data)
}

func (u *Updater) routine() {
  for {
    select {
    case local_bundle := <-u.Local_bundles:
      // TODO: Check that the local bundle is in bounds
      u.local_frame = local_bundle.Frame
      for frame := u.global_frame + 1; frame <= u.local_frame; frame++ {
        u.initFrameData(frame)
      }
      if u.global_frame < u.local_frame {
        u.global_frame = u.local_frame
      }
      if u.local_frame < u.oldest_dirty_frame {
        println(u.Params.Id, "Reset oldest from ", u.oldest_dirty_frame, " to ", u.local_frame)
        u.oldest_dirty_frame = u.local_frame
      }
      data := u.data_window.Get(local_bundle.Frame)
      data.Bundle.AbsorbEventBundle(local_bundle.Bundle)
      u.data_window.Set(local_bundle.Frame, data)
      u.Broadcast_bundles <- local_bundle
      u.advance()

    case remote_bundle := <-u.Remote_bundles:
      for frame := u.global_frame + 1; frame <= remote_bundle.Frame; frame++ {
        u.initFrameData(frame)
      }
      if u.global_frame < remote_bundle.Frame {
        u.global_frame = remote_bundle.Frame
      }
      if remote_bundle.Frame < u.oldest_dirty_frame {
        println(u.Params.Id, "Reset oldest from ", u.oldest_dirty_frame, " to ", remote_bundle.Frame)
        u.oldest_dirty_frame = remote_bundle.Frame
      }
      // TODO: Check that the remote bundle is in bounds
      data := u.data_window.Get(remote_bundle.Frame)
      data.Bundle.AbsorbEventBundle(remote_bundle.Bundle)
      u.data_window.Set(remote_bundle.Frame, data)
      u.advance()

    case req := <-u.request_state:
      if req {
        g := u.data_window.Get(u.data_window.Start()).Game
        u.current_state <- g
      } else {
        u.current_state <- u.data_window.Get(u.global_frame).Game
      }

    case <-u.shutdown:
      close(u.Broadcast_bundles)
      return
    }
  }
}

func (u *Updater) RequestFinalGameState() Game {
  u.request_state <- true
  return <-u.current_state
}

func (u *Updater) RequestFastGameState() Game {
  u.request_state <- false
  return <-u.current_state
}

func (u *Updater) Shutdown() {
  u.shutdown <- struct{}{}
}
