package app

import (
	"github.com/grizzlybite/gonsul/internal/config"
	"github.com/grizzlybite/gonsul/internal/util"

	"context"
	"fmt"
	"time"
)

type Ipoll interface {
	RunPoll(ctx context.Context) error
}

type poll struct {
	config     config.IConfig
	logger     util.ILogger
	once       Ionce
	iterations int
}

func NewPoll(config config.IConfig, logger util.ILogger, once Ionce, it int) Ipoll {
	return &poll{
		config:     config,
		logger:     logger,
		once:       once,
		iterations: it,
	}
}

func (a *poll) RunPoll(ctx context.Context) error {
	a.logger.PrintInfo("Starting in mode: POLL")

	// Loop forever
	count := 1
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		a.logger.PrintDebug(fmt.Sprintf("POLL: performing iteration %d", count))
		// Run our once step
		if err := a.once.RunOnce(ctx); err != nil {
			if util.ErrorCode(err, 0) != util.ErrorDeleteNotAllowed {
				return err
			}
			a.logger.PrintError(err.Error())
		}

		timer := time.NewTimer(time.Second * time.Duration(a.config.GetPollInterval()))
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}

		// Make sure we respect the give max iterations (zero means infinite loop)
		// NOTE: This is only useful for testing purposes
		if a.iterations > 0 && a.iterations == count {
			break
		}

		// Increment our iteration counter
		count++
	}

	return nil
}
