package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/config"
)

type MockLegalHoldJob struct {
	mock.Mock
}

func (m *MockLegalHoldJob) GetID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockLegalHoldJob) OnConfigurationChange(cfg *config.Configuration) error {
	args := m.Called(cfg)
	return args.Error(0)
}

func (m *MockLegalHoldJob) Stop(timeout time.Duration) error {
	args := m.Called(timeout)
	return args.Error(0)
}

func (m *MockLegalHoldJob) RunAll() {
	m.Called()
}

func (m *MockLegalHoldJob) RunSingleLegalHold(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockLegalHoldJob) GetRunningLegalHolds() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func setupTestPlugin(t *testing.T) (*Plugin, *plugintest.API) {
	t.Helper()
	p := &Plugin{}
	api := &plugintest.API{}
	p.SetDriver(&plugintest.Driver{})
	p.SetAPI(api)
	p.Client = pluginapi.NewClient(p.API, p.Driver)
	return p, api
}

func TestServeHTTPAuthorization(t *testing.T) {
	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/legalholds"},
		{http.MethodPost, "/api/v1/legalholds"},
		{http.MethodPost, fmt.Sprintf("/api/v1/legalholds/%s/release", model.NewId())},
		{http.MethodPut, fmt.Sprintf("/api/v1/legalholds/%s", model.NewId())},
		{http.MethodGet, fmt.Sprintf("/api/v1/legalholds/%s/download", model.NewId())},
		{http.MethodPost, fmt.Sprintf("/api/v1/legalholds/%s/run", model.NewId())},
		{http.MethodPost, "/api/v1/test_amazon_s3_connection"},
		{http.MethodGet, "/api/v1/groups/search"},
		{http.MethodPost, "/api/v1/legalhold/run"},
	}

	t.Run("unauthenticated user is rejected", func(t *testing.T) {
		for _, ep := range endpoints {
			t.Run(fmt.Sprintf("%s %s", ep.method, ep.path), func(t *testing.T) {
				p, _ := setupTestPlugin(t)

				req, err := http.NewRequest(ep.method, ep.path, nil)
				require.NoError(t, err)
				// No Mattermost-User-ID header set

				recorder := httptest.NewRecorder()
				p.ServeHTTP(nil, recorder, req)

				require.Equal(t, http.StatusUnauthorized, recorder.Code)
				require.Equal(t, "Not authorized\n", recorder.Body.String())
			})
		}
	})

	t.Run("non-admin user is rejected", func(t *testing.T) {
		for _, ep := range endpoints {
			t.Run(fmt.Sprintf("%s %s", ep.method, ep.path), func(t *testing.T) {
				p, api := setupTestPlugin(t)

				api.On("HasPermissionTo", "regular_user_id", model.PermissionManageSystem).Return(false)

				req, err := http.NewRequest(ep.method, ep.path, nil)
				require.NoError(t, err)
				req.Header.Set("Mattermost-User-Id", "regular_user_id")

				recorder := httptest.NewRecorder()
				p.ServeHTTP(nil, recorder, req)

				require.Equal(t, http.StatusUnauthorized, recorder.Code)
				require.Equal(t, "Not authorized\n", recorder.Body.String(),
					"response body must only contain the error message, no leaked data")
			})
		}
	})
}

func TestRunSingleLegalHold(t *testing.T) {
	p := &Plugin{}
	api := &plugintest.API{}
	p.SetDriver(&plugintest.Driver{})
	p.SetAPI(api)
	p.Client = pluginapi.NewClient(p.API, p.Driver)

	api.On("HasPermissionTo", "test_user_id", model.PermissionManageSystem).Return(true)
	api.On("LogInfo", mock.Anything).Maybe()
	api.On("LogError", mock.Anything, mock.Anything).Maybe()

	// Mock the legal hold job
	mockJob := &MockLegalHoldJob{}
	p.legalHoldJob = mockJob

	// Test successful run
	testID := model.NewId()
	mockJob.On("RunSingleLegalHold", testID).Return(nil).Once()

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/legalholds/%s/run", testID), nil)
	require.NoError(t, err)
	req.Header.Add("Mattermost-User-Id", "test_user_id")

	recorder := httptest.NewRecorder()
	p.ServeHTTP(nil, recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)

	// Test invalid legal hold ID
	req, err = http.NewRequest(http.MethodPost, "/api/v1/legalholds/invalid_id/run", nil)
	require.NoError(t, err)
	req.Header.Add("Mattermost-User-Id", "test_user_id")

	recorder = httptest.NewRecorder()
	p.ServeHTTP(nil, recorder, req)
	require.Equal(t, http.StatusNotFound, recorder.Code)

	// Test error running legal hold
	lhID := model.NewId()
	mockJob.On("RunSingleLegalHold", lhID).Return(fmt.Errorf("test error")).Once()

	req, err = http.NewRequest(http.MethodPost, "/api/v1/legalholds/"+lhID+"/run", nil)
	require.NoError(t, err)
	req.Header.Add("Mattermost-User-Id", "test_user_id")

	recorder = httptest.NewRecorder()
	p.ServeHTTP(nil, recorder, req)
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestTestAmazonS3Connection(t *testing.T) {
	p := &Plugin{}
	api := &plugintest.API{}
	p.SetDriver(&plugintest.Driver{})
	p.SetAPI(api)
	p.Client = pluginapi.NewClient(p.API, p.Driver)

	api.On("HasPermissionTo", "test_user_id", model.PermissionManageSystem).Return(true)
	api.On("LogInfo", mock.Anything).Maybe()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bucket/" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	defer server.Close()

	p.setConfiguration(&config.Configuration{
		TimeOfDay: "10:00pm -0500h",
		AmazonS3BucketSettings: config.AmazonS3BucketSettings{
			Enable: true,
			Settings: model.FileSettings{
				DriverName:                         model.NewString("amazons3"),
				AmazonS3Bucket:                     model.NewString("bucket"),
				AmazonS3AccessKeyId:                model.NewString("access_key_id"),
				AmazonS3SecretAccessKey:            model.NewString("secret_access_key"),
				AmazonS3RequestTimeoutMilliseconds: model.NewInt64(5000),
				AmazonS3Endpoint:                   model.NewString(server.Listener.Addr().String()),
				AmazonS3Region:                     model.NewString("us-east-1"),
				AmazonS3SSL:                        model.NewBool(false),
				AmazonS3SSE:                        model.NewBool(false),
			},
		},
	})

	req, err := http.NewRequest(http.MethodPost, "/api/v1/test_amazon_s3_connection", nil)
	require.NoError(t, err)

	req.Header.Add("Mattermost-User-Id", "test_user_id")

	recorder := httptest.NewRecorder()
	p.ServeHTTP(nil, recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)
}
