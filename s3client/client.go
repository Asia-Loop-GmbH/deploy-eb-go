package s3client

import (
	"bytes"
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io/ioutil"
	"log"
)

type S3Client struct {
	client *s3.Client
}

func NewS3Client(cfg aws.Config) *S3Client {
	return &S3Client{s3.NewFromConfig(cfg)}
}

func (s3Client *S3Client) Upload(filePath, bucket, key string) {
	log.Printf("upload '%s' to '%s' in bucket '%s'", filePath, key, bucket)

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err)
	}
	reader := bytes.NewReader(data)

	_, err = s3Client.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   reader,
	})

	if err != nil {
		panic(err)
	}

	log.Println("upload succeeded")
}
