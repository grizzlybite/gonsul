package mocks

import (
	"context"
	"net/http"

	"github.com/stretchr/testify/mock"
)

type IConfig struct {
	mock.Mock
}

func (m *IConfig) IsCloning() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *IConfig) GetLogLevel() int {
	args := m.Called()
	return args.Int(0)
}

func (m *IConfig) GetStrategy() string {
	args := m.Called()
	return args.String(0)
}

func (m *IConfig) GetRepoURL() string {
	args := m.Called()
	return args.String(0)
}

func (m *IConfig) GetRepoSSHKey() string {
	args := m.Called()
	return args.String(0)
}

func (m *IConfig) GetRepoSSHUser() string {
	args := m.Called()
	return args.String(0)
}

func (m *IConfig) GetRepoBranch() string {
	args := m.Called()
	return args.String(0)
}

func (m *IConfig) GetRepoRemoteName() string {
	args := m.Called()
	return args.String(0)
}

func (m *IConfig) GetRepoBasePath() string {
	args := m.Called()
	return args.String(0)
}

func (m *IConfig) GetRepoRootDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *IConfig) GetConsulURL() string {
	args := m.Called()
	return args.String(0)
}

func (m *IConfig) GetConsulACL() string {
	args := m.Called()
	return args.String(0)
}

func (m *IConfig) GetConsulBasePath() string {
	args := m.Called()
	return args.String(0)
}

func (m *IConfig) ShouldExpandJSON() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *IConfig) ShouldExpandYAML() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *IConfig) DoSecrets() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *IConfig) GetSecretsMap() map[string]string {
	args := m.Called()
	if value, ok := args.Get(0).(map[string]string); ok {
		return value
	}
	return nil
}

func (m *IConfig) AllowDeletes() string {
	args := m.Called()
	return args.String(0)
}

func (m *IConfig) GetPollInterval() int {
	args := m.Called()
	return args.Int(0)
}

func (m *IConfig) WorkingChan() chan bool {
	args := m.Called()
	if value, ok := args.Get(0).(chan bool); ok {
		return value
	}
	return nil
}

func (m *IConfig) GetValidExtensions() []string {
	args := m.Called()
	if value, ok := args.Get(0).([]string); ok {
		return value
	}
	return nil
}

func (m *IConfig) KeepFileExt() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *IConfig) GetTimeout() int {
	args := m.Called()
	return args.Int(0)
}

func (m *IConfig) GetHookAddr() string {
	args := m.Called()
	return args.String(0)
}

func (m *IConfig) GetDryRunOutput() string {
	args := m.Called()
	return args.String(0)
}

func (m *IConfig) IsShowVersion() bool {
	args := m.Called()
	return args.Bool(0)
}

type ILogger struct {
	mock.Mock
}

func (m *ILogger) PrintError(msg string) {
	m.Called(msg)
}

func (m *ILogger) PrintInfo(msg string) {
	m.Called(msg)
}

func (m *ILogger) PrintDebug(msg string) {
	m.Called(msg)
}

func (m *ILogger) AddMessage(msg string) {
	m.Called(msg)
}

func (m *ILogger) GetMessages() []string {
	args := m.Called()
	if value, ok := args.Get(0).([]string); ok {
		return value
	}
	return nil
}

type IExporter struct {
	mock.Mock
}

func (m *IExporter) Start() (map[string]string, error) {
	args := m.Called()
	if value, ok := args.Get(0).(map[string]string); ok {
		return value, args.Error(1)
	}
	return nil, args.Error(1)
}

type IImporter struct {
	mock.Mock
}

func (m *IImporter) Start(ctx context.Context, localData map[string]string) error {
	args := m.Called(ctx, localData)
	return args.Error(0)
}

type Ionce struct {
	mock.Mock
}

func (m *Ionce) RunOnce(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type Ihook struct {
	mock.Mock
}

func (m *Ihook) RunHook(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type Ipoll struct {
	mock.Mock
}

func (m *Ipoll) RunPoll(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type IHookHttp struct {
	mock.Mock
}

func (m *IHookHttp) Start(ctx context.Context, route string, handler func(http.ResponseWriter, *http.Request)) error {
	args := m.Called(ctx, route, handler)
	return args.Error(0)
}
