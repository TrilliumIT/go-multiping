package ticker

import "context"

// Ticker is a ticker for an ICMP sequence
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

func (ft *FloodTicker) Ready() {
	ft.ready <- struct{}{}
	ft.ManualTicker.Ready()
}

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
