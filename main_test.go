package main

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/gambol99/go-marathon"
	"time"
	"fmt"
)

type MockClient struct {
	mock.Mock
}

func (m *MockClient) RestartApplication(appID string, force bool) (*marathon.DeploymentID, error) {
	args := m.Called(appID, force)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	result := args.Get(0).(marathon.DeploymentID)
	return &result, args.Error(1)
}

func (m *MockClient) WaitOnDeployment(deployID string, timeout time.Duration) (err error) {
	args := m.Called(deployID, timeout)
	return args.Error(0)
}

var (
	firstAppID  = "/test/first.app"
	secondAppID = "/test/second.app"
	testApps    = []marathon.Application{
		{ID: firstAppID},
		{ID: secondAppID},
	}
	deployOne = marathon.DeploymentID{DeploymentID: firstAppID}
	deployTwo = marathon.DeploymentID{DeploymentID: secondAppID}
)

func TestIfRestartsAllGivenApplications(t *testing.T) {
	mockClient := new(MockClient)

	mockClient.On("RestartApplication", firstAppID, false).Return(deployOne, nil).Once()
	mockClient.On("RestartApplication", secondAppID, false).Return(deployTwo, nil).Once()

	mockClient.On("WaitOnDeployment", firstAppID, timeout).Return(nil).Once()
	mockClient.On("WaitOnDeployment", secondAppID, timeout).Return(nil).Once()

	// when
	remaining := restartApps(testApps, mockClient)

	// then
	require.Empty(t, remaining)
	mockClient.AssertExpectations(t)
}

func TestIfRestartErrorsAreRetryable(t *testing.T) {
	mockClient := new(MockClient)

	mockClient.On("RestartApplication", firstAppID, false).Return(nil, fmt.Errorf("test error")).Once()
	mockClient.On("RestartApplication", secondAppID, false).Return(deployTwo, nil).Once()

	mockClient.On("WaitOnDeployment", secondAppID, timeout).Return(nil).Once()

	// when
	remaining := restartApps(testApps, mockClient)

	// then
	require.NotEmpty(t, remaining)
	require.Contains(t, remaining, testApps[0])
	mockClient.AssertExpectations(t)
}
