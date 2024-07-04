package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/config"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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
