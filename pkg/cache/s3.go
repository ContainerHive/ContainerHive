package cache

import "strconv"

type S3BuildKitCache struct {
	EndpointUrl     string
	Bucket          string
	Region          string
	AccessKeyId     string
	SecretAccessKey string
	UsePathStyle    bool
	CacheKey        string
}

func (s *S3BuildKitCache) Name() string {
	return "s3"
}

func (s *S3BuildKitCache) ToAttributes() map[string]string {
	return map[string]string{
		"endpoint_url":      s.EndpointUrl,
		"bucket":            s.Bucket,
		"region":            s.Region,
		"access_key_id":     s.AccessKeyId,
		"secret_access_key": s.SecretAccessKey,
		"use_path_style":    strconv.FormatBool(s.UsePathStyle),
		"mode":              "max",
		"name":              s.CacheKey,
	}
}
