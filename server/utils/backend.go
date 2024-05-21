package utils

import (
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/shared/filestore"
)

func GetBackendSettings(endPoint string) filestore.FileBackendSettings {
	return filestore.FileBackendSettings{
		DriverName:                         model.ImageDriverS3,
		AmazonS3AccessKeyId:                model.MinioAccessKey,
		AmazonS3SecretAccessKey:            model.MinioSecretKey,
		AmazonS3Bucket:                     model.MinioBucket,
		AmazonS3Region:                     "",
		AmazonS3Endpoint:                   endPoint,
		AmazonS3PathPrefix:                 "",
		AmazonS3SSL:                        false,
		AmazonS3SSE:                        false,
		AmazonS3RequestTimeoutMilliseconds: 5000,
	}
}
