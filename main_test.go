package main

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/gambol99/go-marathon"
	"time"
)

type MockClient struct {
	mock.Mock
}

func (m *MockClient) RestartApplication(appID string, force bool) (*marathon.DeploymentID, error) {
	args := m.Called(appID, force)
	result := args.Get(0).(marathon.DeploymentID)
	return &result, args.Error(1)
}

func (m *MockClient) WaitOnDeployment(deployID string, timeout time.Duration) (err error) {
	args := m.Called(deployID, timeout)
	return args.Error(0)
}

func TestIfRestartsAllGivenApplications(t *testing.T) {
	mockClient := new(MockClient)

	firstAppID := "/test/first.app"
	secondAppID := "/test/second.app"
	timeout := 120 * time.Second
	apps := []marathon.Application{
		{ID: firstAppID},
		{ID: secondAppID},
	}
	deployOne := marathon.DeploymentID{DeploymentID: firstAppID}
	deployTwo := marathon.DeploymentID{DeploymentID: secondAppID}

	mockClient.On("RestartApplication", firstAppID, false).Return(deployOne, nil).Once()
	mockClient.On("RestartApplication", secondAppID, false).Return(deployTwo, nil).Once()

	mockClient.On("WaitOnDeployment", firstAppID, timeout).Return(nil).Once()
	mockClient.On("WaitOnDeployment", secondAppID,timeout).Return(nil).Once()

	// when
	remaining := restartApps(apps, mockClient)

	// then
	require.Empty(t, remaining)
	mockClient.AssertExpectations(t)
}
