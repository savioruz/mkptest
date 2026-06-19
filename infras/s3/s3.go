package s3

//go:generate go run go.uber.org/mock/mockgen -source=./s3.go -destination=./mocks/s3_mock.go -package=mocks

import (
	"bytes"
	"context"
	"fmt"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"

	"mime/multipart"
	"oil/config"
	"oil/infras/otel"
	"oil/shared/constant"
)

const (
	otelAttrFileName = "file_name"
	otelAttrBucket   = "bucket"
)

type S3 interface {
	UploadFile(ctx context.Context, bucketName, directory string, file multipart.File, fileHeader *multipart.FileHeader, fileName string) (url string, err error)
	UploadFileBytes(ctx context.Context, bucketName, directory, fileName, contentType string, fileData []byte) (url string, err error)
	DeleteFile(ctx context.Context, bucketName, directory, objectName string) error
	GetObjectNameFromURL(bucketName, url string) (objectName string)
}

type s3Impl struct {
	Client *s3.Client
	Config *config.Config
	otel   otel.Otel
}

func (svc *s3Impl) UploadFile(ctx context.Context, bucketName, directory string, file multipart.File, fileHeader *multipart.FileHeader, fileName string) (url string, err error) {
	ctx, scope := svc.otel.NewScope(ctx, constant.OtelS3ScopeName, constant.OtelS3ScopeName+".UploadFile")
	defer scope.End()
	defer scope.TraceIfError(err)

	if bucketName == "" {
		bucketName = svc.Config.External.S3.BucketName
	}

	scope.SetAttributes(map[string]any{
		otelAttrFileName: fileName,
		otelAttrBucket:   bucketName,
	})

	buf := bytes.NewBuffer(nil)

	if _, err = buf.ReadFrom(file); err != nil {
		return constant.Empty, fmt.Errorf("failed to read file: %w", err)
	}

	contentType := fileHeader.Header.Get(constant.RequestHeaderContentType)

	return svc.upload(ctx, bucketName, directory, fileName, contentType, buf)
}

func (svc *s3Impl) UploadFileBytes(ctx context.Context, bucketName, directory, fileName, contentType string, fileData []byte) (url string, err error) {
	ctx, scope := svc.otel.NewScope(ctx, constant.OtelS3ScopeName, constant.OtelS3ScopeName+".UploadFileBytes")
	defer scope.End()
	defer scope.TraceIfError(err)

	if bucketName == "" {
		bucketName = svc.Config.External.S3.BucketName
	}

	scope.SetAttributes(map[string]any{
		otelAttrFileName: fileName,
		otelAttrBucket:   bucketName,
	})

	buf := bytes.NewBuffer(fileData)

	return svc.upload(ctx, bucketName, directory, fileName, contentType, buf)
}

func (svc *s3Impl) DeleteFile(ctx context.Context, bucketName, directory, objectName string) (err error) {
	ctx, scope := svc.otel.NewScope(ctx, constant.OtelS3ScopeName, constant.OtelS3ScopeName+".DeleteFile")
	defer scope.End()
	defer scope.TraceIfError(err)

	scope.SetAttributes(map[string]any{
		otelAttrFileName: objectName,
		otelAttrBucket:   bucketName,
	})

	objectKey := path.Join(directory, objectName)

	_, err = svc.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to delete file from S3")

		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	return nil
}

func (svc *s3Impl) GetObjectNameFromURL(bucketName, url string) (objectName string) {
	publicDomain := svc.Config.External.S3.PublicDomain

	bucketPrefix := path.Join(publicDomain, bucketName) + "/"
	if len(url) >= len(bucketPrefix) && url[:len(bucketPrefix)] == bucketPrefix {
		return url[len(bucketPrefix):]
	}

	apiEndpoint := svc.Config.External.S3.APIEndpoint

	bucketURL := fmt.Sprintf("%s/%s/", apiEndpoint, bucketName)
	if len(url) >= len(bucketURL) && url[:len(bucketURL)] == bucketURL {
		return url[len(bucketURL):]
	}

	return constant.Empty
}

func (svc *s3Impl) upload(ctx context.Context, bucket, directory, fileName, contentType string, buf *bytes.Buffer) (url string, err error) {
	ctx, scope := svc.otel.NewScope(ctx, constant.OtelS3ScopeName, constant.OtelS3ScopeName+".upload")
	defer scope.End()
	defer scope.TraceIfError(err)

	objectKey := path.Join(directory, fileName)
	fileReader := bytes.NewReader(buf.Bytes())

	_, err = svc.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(objectKey),
		Body:          fileReader,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(fileReader.Size()),
	})
	if err != nil {
		return constant.Empty, fmt.Errorf("failed to upload file to S3: %w", err)
	}

	publicDomain := svc.Config.External.S3.PublicDomain

	return fmt.Sprintf("%s/%s", publicDomain, objectKey), nil
}

func New(config *config.Config, otel otel.Otel) S3 {
	endpoint := config.External.S3.APIEndpoint
	accessKeyID := config.External.S3.AccessKeyID
	secretAccessKey := config.External.S3.SecretAccessKey

	staticProvider := credentials.NewStaticCredentialsProvider(
		accessKeyID,
		secretAccessKey,
		"",
	)

	cfg, err := awsConfig.LoadDefaultConfig(
		context.TODO(),
		awsConfig.WithCredentialsProvider(staticProvider),
	)

	if err != nil {
		log.Err(err).Msg("Error loading AWS configuration")
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
		o.Region = "auto"
	})

	return &s3Impl{
		Client: s3Client,
		Config: config,
		otel:   otel,
	}
}
