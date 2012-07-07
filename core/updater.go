package core

// The updater has the following tasks:
// Receive Events from all engines, including localhost, store the events and
// apply them to the Game as necessary.  If events show up late it will rewind
// the Game to an older state and reapply all Events in the proper order.
type Updater struct {
  // EngineBundles formed by this engine.  These will always come in order and
  // have their StateFrame attached, so this channel also serves as a clock.
  // Local_bundles <-chan EngineBundle

  // // FrameBundles 
  // Remote_bundles <-chan FrameBundle
}
