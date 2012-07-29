package core

import (
  "encoding/gob"
)

type EngineId int64
type StateFrame int

type EngineParams struct {
  // Unique Id of this Engine
  Id EngineId

  // Number of frames to wait before applying the events on a GameState
  Delay StateFrame

  // Duration of a StateFrame
  Frame_ms int64

  // Number of frames to keep around in case the Game state needs to be
  // rewinded.  The more frames kept around the more memory is required, but
  // also the more latency that can be tolerated before pausing the game or
  // dropping players.
  Max_frames int
}
type Engine struct {
  network Network
  params  EngineParams
}

type EngineEvent interface {
  Apply(*EngineInfo)
}

func init() {
  gob.Register(EngineJoined{})
  gob.Register(EngineDropped{})
}

type EngineJoined struct {
  Id EngineId
}

func (e EngineJoined) Apply(info *EngineInfo) {
  info.Engines[e.Id] = true
}

type EngineDropped struct {
  Id EngineId
}

func (e EngineDropped) Apply(info *EngineInfo) {
  delete(info.Engines, e.Id)
}

// Contains information necessary to processing StateFrames.  The data in an
// EngineInfo can also be modified, like the GameState, but can only be done
// by the host.
type EngineInfo struct {
  // Set of all known engines on this StateFrame.  Events must be received
  // from each known engine on the next frame for the next frame to complete.
  // This means that no events are expected from an engine on the first frame
  // on which that engine is listed in this set.
  Engines map[EngineId]bool
}

func (ei *EngineInfo) Copy() EngineInfo {
  var ei2 EngineInfo
  ei2.Engines = make(map[EngineId]bool)
  for k, v := range ei.Engines {
    ei2.Engines[k] = v
  }
  return ei2
}

type FrameData struct {
  Info   EngineInfo
  Game   Game
  Bundle EventBundle
}
