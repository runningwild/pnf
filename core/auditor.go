package core

// Data from remote connections comes to the Auditor from the Communicator.
// The Auditor verifies remote information and sends it to the Updater, if any
// connections are dropped the Auditor will create dummy bundles so that the
// game can continue without those connections, and will also signal to other
// engines that no more bundles are expected from those engines.
type Auditor struct {
  // Bundles come here from the Communicator.  At this point they have not
  // been verified.
  Raw_remote_bundles <-chan FrameBundle

  // After bundles have been verified they are sent through here to the
  // Updater.  Dummy events are also sent through here.
  Remote_bundles chan<- FrameBundle

  // If the Auditor detects that this engine is out of sync with other engines
  // it can tell the Bundler so that it can adjust its clock accordingly.
  Time_delta chan<- int64
}

func (a *Auditor) Start() {
  go a.routine()
}

// TODO: Currently this is just a pass-through auditor, the following things
// must be implemented for a robust engine:
// - Detect dropped players, probably need to be notified by the Communicator,
//   and generate dummy events and a PlayerLeft engine event.
// - Detect when we're not synchronized and adjust clocks accordingly.
func (a *Auditor) routine() {
  for {
    select {
    case raw_remote := <-a.Raw_remote_bundles:
      a.Remote_bundles <- raw_remote
    }
  }
}
