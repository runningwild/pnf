package pnf

type EngineId int64
type StateFrame int

type EngineParams struct {
  delay_ms   int
  frame_ms   int
  max_frames int
}
type Engine struct {
  id      EngineId
  network Network

  params EngineParams
  states GameWindow
  events EventsWindow
}

func NewEngine(network Network) *Engine {
  return &Engine{
    id:      EngineId(RandomId()),
    network: network,
  }
}

func (e *Engine) StartNewGame(params EngineParams, game Game) {
  e.params = params
  e.states.Set(0, game)
}

// Queues the event to be applied on a future game state that is at least
// Delay frames in the future.
func (e *Engine) ApplyEvent(Event) {}

func (e *Engine) GetState(timestep int, game *Game) {}
