package utils

import (
	"blog/config"
	"context"
	"errors"
	"fmt"
	"mime/multipart"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

// 只在第一次运行时给client赋值
var client *oss.Client
var ossConfig config.OssConfig

// 上传图片
func UploadToOss(body multipart.File, path string, filename string) (string, error) {
	if client == nil || ossConfig.AccessKeyId == "" {
		if err := initOssClient(); err != nil {
			return "", err
		}
	}
	key := path + filename
	request := &oss.PutObjectRequest{
		Bucket: oss.Ptr(ossConfig.Bucket),
		Key:    oss.Ptr(key),
		Body:   body,
	}
	if _, err := client.PutObject(context.TODO(), request); err != nil {
		return "", err
	}
	return fmt.Sprintf("https://%s.oss-%s.aliyuncs.com/%s", ossConfig.Bucket, ossConfig.Endpoint, key), nil
}

// 初始化ossConfig和client
func initOssClient() error {
	ossConfig = config.Cfg.OssConfig
	if ossConfig.AccessKeyId == "" || ossConfig.AccessKeySecret == "" || ossConfig.Bucket == "" || ossConfig.Endpoint == "" {
		return errors.New("oss config is incomplete")
	}
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			ossConfig.AccessKeyId,
			ossConfig.AccessKeySecret,
			"",
		)).
		WithRegion(ossConfig.Endpoint)
	client = oss.NewClient(cfg)
	return nil
}
