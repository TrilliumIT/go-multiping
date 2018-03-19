package ticker

import "context"

// FloodTicker is a ticker that sends packets as fast as they are recieved
// C will fire either on the interval, or as soon as Cont is called followed by wait not blocking
type FloodTicker struct {
	ManualTicker
	ready chan struct{}
	wait  func()
}

// NewFloodTicker reutrns a new flood ticker.
// Wait should be a function that will block the next tick from firing. For example: A waitgroup for pending packets.
// Ready must be triggered before the flood ping will wait on wait(). This is necessary to prevent a race between
// waiting and adding to a waitgroup
func NewFloodTicker(wait func()) Ticker {
	return &FloodTicker{
		newManualTicker(),
		make(chan struct{}),
		wait,
	}
}

// Ready informs the ticker that the wait function is ready to be waited on without racing
func (ft *FloodTicker) Ready() {
	ft.ready <- struct{}{}
	ft.ManualTicker.Ready()
}

// Run runs the flood ticker
func (ft *FloodTicker) Run(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ft.ready:
			}

			ft.wait()
			select {
			case <-ctx.Done():
				return
			case ft.ManualTicker.tick <- struct{}{}:
			}
		}
	}()
	ft.ManualTicker.Run(ctx)
}
