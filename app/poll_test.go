package app

import (
	"fmt"
	"github.com/grizzlybite/gonsul/tests/mocks"

	"context"
	"github.com/grizzlybite/gonsul/internal/util"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"testing"
)

func TestPoll_RunPoll(t *testing.T) {
	RegisterTestingT(t)

	// Create our mocks, our Once mode and our application
	cfg, log, _, _ := getCommonMocks()
	once := &mocks.Ionce{}
	poll := getMockedPoll(cfg, log, once)

	// Create our assertions
	cfg.On("GetPollInterval").Return(1)
	log.On("PrintInfo", mock.Anything).Return()
	log.On("PrintDebug", mock.Anything).Return()
	once.On("RunOnce", mock.Anything).Return(nil)

	// Run our application mode
	err := poll.RunPoll(context.Background())
	Expect(err).NotTo(HaveOccurred())

	// Create our expectations
	Expect(cfg.AssertExpectations(t)).To(BeTrue(), "Assert GetPollInterval")
	Expect(cfg.AssertNumberOfCalls(t, "GetPollInterval", 1))

	Expect(log.AssertExpectations(t)).To(BeTrue(), "Assert Logger")

	Expect(once.AssertExpectations(t)).To(BeTrue(), "Assert Once Run")
	Expect(once.AssertNumberOfCalls(t, "RunOnce", 1))
}

func TestPoll_RunPollContinuesOnDeleteNotAllowed(t *testing.T) {
	RegisterTestingT(t)

	cfg, log, _, _ := getCommonMocks()
	once := &mocks.Ionce{}
	poll := getMockedPoll(cfg, log, once)

	cfg.On("GetPollInterval").Return(0)
	log.On("PrintInfo", mock.Anything).Return()
	log.On("PrintDebug", mock.Anything).Return()
	log.On("PrintError", mock.Anything).Return()
	once.On("RunOnce", mock.Anything).Return(util.NewGonsulError(fmt.Errorf("deletes are not allowed"), util.ErrorDeleteNotAllowed))

	err := poll.RunPoll(context.Background())
	Expect(err).NotTo(HaveOccurred())

	Expect(once.AssertNumberOfCalls(t, "RunOnce", 1))
	Expect(log.AssertCalled(t, "PrintError", "deletes are not allowed")).To(BeTrue())
}
