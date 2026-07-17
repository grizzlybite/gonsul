package app

import (
	"github.com/grizzlybite/gonsul/internal/config"

	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// Application ...
type Application struct {
	config  config.IConfig
	once    Ionce
	hook    Ihook
	poll    Ipoll
	sigChan chan os.Signal
}

// NewApplication ...
func NewApplication(
	config config.IConfig,
	once Ionce,
	hook Ihook,
	poll Ipoll,
	sigChan chan os.Signal,
) *Application {
	return &Application{
		config:  config,
		once:    once,
		hook:    hook,
		poll:    poll,
		sigChan: sigChan,
	}
}

// Start ...
func (a *Application) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Relay all Signals to our channel
	signal.Notify(a.sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Spin a routine to wait for a Signal
	go func() {
		// Wait for a signal through the channel
		<-a.sigChan
		cancel()
		fmt.Print(" Interrupt received, waiting for work to finish... ")
		// Try to write to working channel (thus waiting for any in progress non interruptible work)
		a.config.WorkingChan() <- false
		fmt.Print(" Quitting!")
	}()

	// Switch our run strategy
	switch a.config.GetStrategy() {
	case config.StrategyDry, config.StrategyOnce:
		return a.once.RunOnce(ctx)
	case config.StrategyHook:
		return a.hook.RunHook(ctx)
	case config.StrategyPoll:
		return a.poll.RunPoll(ctx)
	}

	return nil
}
