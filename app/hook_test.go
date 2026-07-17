package app

import (
	"net/http"
	"net/http/httptest"

	"fmt"
	"github.com/grizzlybite/gonsul/internal/util"
	"github.com/grizzlybite/gonsul/tests/mocks"

	"context"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"testing"
)

func TestHook_RunHook(t *testing.T) {
	RegisterTestingT(t)

	// Create our mocks and our Once mode
	cfg, log, _, _ := getCommonMocks()
	http := &mocks.IHookHttp{}
	once := &mocks.Ionce{}
	hook := getMockedHook(http, cfg, log, once)

	// Create our assertions
	http.On("Start", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	log.On("PrintInfo", mock.Anything).Return()

	// Run our application mode
	err := hook.RunHook(context.Background())
	Expect(err).NotTo(HaveOccurred())

	// Create our expectations
	Expect(http.AssertExpectations(t)).To(BeTrue(), "Assert Http.Start")
	Expect(http.AssertNumberOfCalls(t, "Start", 1))
	Expect(log.AssertExpectations(t)).To(BeTrue(), "Assert Logger")
}

func TestHook_HTTPHandlerReturnsServiceUnavailableOnError(t *testing.T) {
	RegisterTestingT(t)

	cfg, log, _, _ := getCommonMocks()
	httpServer := &mocks.IHookHttp{}
	once := &mocks.Ionce{}
	hookApp := NewHook(httpServer, cfg, log, once)

	log.On("PrintInfo", mock.Anything).Return()
	log.On("GetMessages").Return([]string{"stale/key"})
	once.On("RunOnce", mock.Anything).Return(util.NewGonsulError(fmt.Errorf("deletes are not allowed"), util.ErrorDeleteNotAllowed))

	request := httptest.NewRequest(http.MethodGet, "/v1/run", nil)
	response := httptest.NewRecorder()

	hookApp.(*hook).httpHandler(response, request)

	Expect(response.Code).To(Equal(http.StatusServiceUnavailable))
	Expect(response.Header().Get("X-Gonsul-Error")).To(Equal("10"))
	Expect(response.Header().Get("X-Gonsul-Delete-Paths")).To(Equal("stale/key"))
}
