package app

import (
	"github.com/grizzlybite/gonsul/internal/config"
	"github.com/grizzlybite/gonsul/internal/util"

	"context"
	"fmt"
	"net/http"
	"time"
)

// IHookHttp is our interface used for the Hook strategy.
type IHookHttp interface {
	Start(ctx context.Context, route string, handler func(http.ResponseWriter, *http.Request)) error
}

// hookHttp is our IHookHttp concrete implementation
type hookHttp struct {
	config config.IConfig
	logger util.ILogger
}

// NewHookHttp is our hookHttp constructor
func NewHookHttp(config config.IConfig, logger util.ILogger) IHookHttp {
	return &hookHttp{config: config, logger: logger}
}

// Start starts our HTTP server
func (h *hookHttp) Start(ctx context.Context, route string, handler func(http.ResponseWriter, *http.Request)) error {
	// Create our routes and set handlers
	mux := http.NewServeMux()
	mux.HandleFunc(route, handler)

	server := &http.Server{
		Addr:    h.config.GetHookAddr(),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			h.logger.PrintError(fmt.Sprintf("Hook: failed graceful shutdown: %s", err))
		}
	}()

	// Launch our HTTP server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return util.NewGonsulError(fmt.Errorf("Hook: %w", err), util.ErrorFailedHTTPServer)
	}

	return nil
}
