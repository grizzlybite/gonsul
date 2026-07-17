package app

import (
	"github.com/grizzlybite/gonsul/internal/config"
	"github.com/grizzlybite/gonsul/internal/exporter"
	"github.com/grizzlybite/gonsul/internal/importer"
	"github.com/grizzlybite/gonsul/internal/util"

	"context"
)

type Ionce interface {
	RunOnce(ctx context.Context) error
}

type once struct {
	config   config.IConfig
	logger   util.ILogger
	exporter exporter.IExporter
	importer importer.IImporter
}

func NewOnce(config config.IConfig, logger util.ILogger, exporter exporter.IExporter, importer importer.IImporter) Ionce {
	return &once{
		config:   config,
		logger:   logger,
		exporter: exporter,
		importer: importer,
	}
}

// RunOnce is our entry point function for the Once Application mode
func (a *once) RunOnce(ctx context.Context) error {
	strategy := a.config.GetStrategy()
	// Check strategy
	if strategy == config.StrategyDry {
		a.logger.PrintInfo("Starting in mode: DRYRUN")
	} else if strategy == config.StrategyOnce {
		a.logger.PrintInfo("Starting in mode: ONCE")
	}

	// Start our data export
	a.logger.PrintDebug("Starting data retrieve from GIT")
	exportedData, err := a.exporter.Start()
	if err != nil {
		return err
	}
	a.logger.PrintDebug("Finished data retrieve from GIT")

	// Start data import to Consul
	a.logger.PrintDebug("Starting data import to Consul")
	if err := a.importer.Start(ctx, exportedData); err != nil {
		return err
	}
	a.logger.PrintDebug("Finished data import to Consul")
	return nil
}
