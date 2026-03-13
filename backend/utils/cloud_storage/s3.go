package cloud_storage

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	client   *s3.Client
	bucket   string
	basePath string
}

func NewS3Client(region, endpoint, bucket, accessKey, secretKey, basePath string) (*S3Client, error) {
	cfg := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	}

	opts := func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
		o.UsePathStyle = true
	}

	client := s3.NewFromConfig(cfg, opts)
	return &S3Client{client: client, bucket: bucket, basePath: basePath}, nil
}

func (c *S3Client) Upload(src, target string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()
	key := c.basePath + "/" + target
	_, err = c.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	return err
}

func (c *S3Client) Download(src, target string) error {
	key := c.basePath + "/" + src
	resp, err := c.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	defer file.Close()
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			file.Write(buf[:n])
		}
		if readErr != nil {
			break
		}
	}
	return nil
}

func (c *S3Client) Delete(path string) error {
	key := c.basePath + "/" + path
	_, err := c.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (c *S3Client) ListObjects(prefix string) ([]string, error) {
	key := c.basePath + "/" + prefix
	resp, err := c.client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	var result []string
	for _, obj := range resp.Contents {
		result = append(result, *obj.Key)
	}
	return result, nil
}
