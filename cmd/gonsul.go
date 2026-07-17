package main

import (
	"github.com/grizzlybite/gonsul/app"
	"github.com/grizzlybite/gonsul/internal/config"
	"github.com/grizzlybite/gonsul/internal/exporter"
	"github.com/grizzlybite/gonsul/internal/importer"
	"github.com/grizzlybite/gonsul/internal/util"

	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	os.Exit(run())
}

func run() int {
	// Build our configuration
	cfg, err := config.NewConfig()
	if err != nil {
		logger := util.NewLogger(0)
		logger.PrintError(err.Error())
		return util.ErrorBadParams
	}

	// Build our logger
	logger := util.NewLogger(cfg.GetLogLevel())

	// Are we just printing the app version
	if cfg.IsShowVersion() {
		fmt.Println("Gonsul version: " + app.Version)
		fmt.Println("Build date: " + app.BuildDate)
		return 0
	}

	// Build all dependencies for our application
	hookHttpServer := app.NewHookHttp(cfg, logger)
	httpClient := &http.Client{Timeout: time.Second * time.Duration(cfg.GetTimeout())}
	exp := exporter.NewExporter(cfg, logger)
	imp := importer.NewImporter(cfg, logger, httpClient)
	sigChannel := make(chan os.Signal)
	// Build our Applications
	once := app.NewOnce(cfg, logger, exp, imp)
	hook := app.NewHook(hookHttpServer, cfg, logger, once)
	poll := app.NewPoll(cfg, logger, once, 0)
	// Build our main Application container
	application := app.NewApplication(cfg, once, hook, poll, sigChannel)

	// Start our application
	if err := application.Start(); err != nil {
		logger.PrintError(err.Error())
		return util.ErrorCode(err, util.ErrorFailedConsulConnection)
	}

	// We're still here, all went well, good bye
	logger.PrintInfo("Quitting... bye.")
	return 0
}
