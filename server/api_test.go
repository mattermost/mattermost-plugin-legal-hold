package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	mattermostModel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/config"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store/kvstore"
)

func TestTestAmazonS3Connection(t *testing.T) {
	p := &Plugin{}
	api := &plugintest.API{}
	p.SetDriver(&plugintest.Driver{})
	p.SetAPI(api)
	p.Client = pluginapi.NewClient(p.API, p.Driver)

	api.On("HasPermissionTo", "test_user_id", mattermostModel.PermissionManageSystem).Return(true)
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
			Settings: mattermostModel.FileSettings{
				DriverName:                         mattermostModel.NewString("amazons3"),
				AmazonS3Bucket:                     mattermostModel.NewString("bucket"),
				AmazonS3AccessKeyId:                mattermostModel.NewString("access_key_id"),
				AmazonS3SecretAccessKey:            mattermostModel.NewString("secret_access_key"),
				AmazonS3RequestTimeoutMilliseconds: mattermostModel.NewInt64(5000),
				AmazonS3Endpoint:                   mattermostModel.NewString(server.Listener.Addr().String()),
				AmazonS3Region:                     mattermostModel.NewString("us-east-1"),
				AmazonS3SSL:                        mattermostModel.NewBool(false),
				AmazonS3SSE:                        mattermostModel.NewBool(false),
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

func TestRunLegalHoldNow(t *testing.T) {
	validID := mattermostModel.NewId()

	tests := []struct {
		name        string
		legalHoldID string
		wantErr     bool
		expectCode  int
	}{
		{
			name:        "Invalid Legal Hold ID",
			legalHoldID: "malformedid",
			wantErr:     true,
			expectCode:  http.StatusBadRequest,
		},
		{
			name:        "Unknown Legal Hold ID",
			legalHoldID: mattermostModel.NewId(),
			wantErr:     true,
			expectCode:  http.StatusInternalServerError,
		},
		// TODO: To test the successful case, we need to put Plugin.LegalHoldJob behind
		//       an interface so that it can be mocked out.
		// {
		//	name:        "Valid Legal Hold",
		//	legalHoldID: validID,
		//	wantErr:     false,
		//	expectCode:  http.StatusOK,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Plugin{}
			api := &plugintest.API{}
			p.SetDriver(&plugintest.Driver{})
			p.SetAPI(api)
			p.Client = pluginapi.NewClient(p.API, p.Driver)
			p.KVStore = kvstore.NewKVStore(p.Client)

			api.On("HasPermissionTo", "test_user_id", mattermostModel.PermissionManageSystem).Return(true)
			api.On("LogInfo", mock.Anything).Maybe()
			api.On("LogError", mock.Anything, mock.Anything).Maybe()
			api.On("KVGet", fmt.Sprintf("kvstore_legal_hold_%s", validID)).
				Return([]uint8("{}"), nil)
			api.On("KVGet", mock.Anything).
				Return(nil, mattermostModel.NewAppError("things", "stuff.stuff", nil, "", http.StatusNotFound))

			recorder := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/legalhold/%s/run", tt.legalHoldID), nil)
			require.NoError(t, err)

			req.Header.Add("Mattermost-User-Id", "test_user_id")

			p.ServeHTTP(nil, recorder, req)
			require.Equal(t, tt.expectCode, recorder.Code)
			if tt.wantErr {
				assert.NotEmpty(t, recorder.Body)
			} else {
				assert.Empty(t, recorder.Body)
			}
		})
	}
}
