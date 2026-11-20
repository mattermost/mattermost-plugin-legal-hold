package config

import "testing"

func TestEffectiveDriver(t *testing.T) {
	cases := []struct {
		name string
		in   AmazonS3BucketSettings
		want string
	}{
		{name: "legacy disabled", in: AmazonS3BucketSettings{}, want: FilestoreDriverServer},
		{name: "legacy s3", in: AmazonS3BucketSettings{Enable: true}, want: FilestoreDriverS3},
		{name: "explicit server", in: AmazonS3BucketSettings{Enable: true, DriverName: FilestoreDriverServer}, want: FilestoreDriverServer},
		{name: "azure", in: AmazonS3BucketSettings{DriverName: FilestoreDriverAzure}, want: FilestoreDriverAzure},
		{name: "local", in: AmazonS3BucketSettings{DriverName: FilestoreDriverLocal}, want: FilestoreDriverLocal},
		{name: "unknown falls back", in: AmazonS3BucketSettings{DriverName: "nfs"}, want: FilestoreDriverServer},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.in.EffectiveDriver(); got != tc.want {
				t.Fatalf("EffectiveDriver() = %q, want %q", got, tc.want)
			}
		})
	}
}
