package app

import (
	"github.com/grizzlybite/gonsul/internal/config"
	"github.com/grizzlybite/gonsul/internal/util"
	"sync"

	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Ihook interface {
	RunHook(ctx context.Context) error
}

type hook struct {
	mutex  sync.Mutex
	http   IHookHttp
	config config.IConfig
	logger util.ILogger
	once   Ionce
}

func NewHook(http IHookHttp, config config.IConfig, logger util.ILogger, once Ionce) Ihook {
	return &hook{
		mutex:  sync.Mutex{},
		http:   http,
		config: config,
		logger: logger,
		once:   once,
	}
}

// hook ...
func (a *hook) RunHook(ctx context.Context) error {
	// User information
	a.logger.PrintInfo("Starting in mode: HOOK")

	// Start our HTTP Server
	return a.http.Start(ctx, "/v1/run", a.httpHandler)
}

// run ...
func (a *hook) httpHandler(response http.ResponseWriter, request *http.Request) {
	// Make sure this is a GET request
	if request.Method != http.MethodGet {
		response.WriteHeader(http.StatusNotFound)
		_, _ = response.Write([]byte("400 - ups, page not found!"))
		return
	}

	// Let's try to get a lock and defer the unlock
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.logger.PrintInfo("HTTP Incoming connection from: " + request.RemoteAddr)

	// On every request, run Once as usual business
	if err := a.once.RunOnce(request.Context()); err != nil {
		errorCode := util.ErrorCode(err, util.ErrorFailedConsulConnection)
		response.Header().Add("X-Gonsul-Error", strconv.Itoa(errorCode))
		if errorCode == util.ErrorDeleteNotAllowed {
			response.Header().Add("X-Gonsul-Delete-Paths", strings.Join(a.logger.GetMessages(), ","))
		}
		response.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprintf(response, "Error: %d\n", errorCode)
		return
	}

	// If here, process ran smooth, return HTTP 200
	response.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(response, "Done")
}
