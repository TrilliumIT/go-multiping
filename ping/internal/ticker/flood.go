package ticker

import "context"

// FloodTicker is a ticker that sends packets as fast as they are recieved
// C will fire either on the interval, or as soon as Cont is called followed by wait not blocking
type FloodTicker struct {
	ManualTicker
	ready chan []func()
}

// NewFloodTicker reutrns a new flood ticker.
func NewFloodTicker() Ticker {
	return &FloodTicker{
		newManualTicker(),
		make(chan []func()),
	}
}

// Ready tells the ticker to wait on any passed in functions, then execute the next tick.
func (ft *FloodTicker) Ready(f ...func()) {
	ft.ready <- f
	ft.ManualTicker.Ready()
}

// Run runs the flood ticker
func (ft *FloodTicker) Run(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case wf := <-ft.ready:
				for _, f := range wf {
					f()
				}
			}

			select {
			case <-ctx.Done():
				return
			case ft.ManualTicker.tick <- struct{}{}:
			}
		}
	}()
	ft.ManualTicker.Run(ctx)
}
