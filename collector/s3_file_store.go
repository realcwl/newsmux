package collector

import (
	"errors"
	"net/http"

	"github.com/Luismorlan/newsmux/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const (
	TestS3Bucket      = "collector-dev-bucket"
	ProdS3ImageBucket = "newsfeed-crawler-image-output"
	ProdS3FileBucket  = "newsfeed-crawler-file-output"
	CouldFrontPrefix  = "https://d20uffqoe1h0vv.cloudfront.net/"
)

type S3FileStore struct {
	bucket                    string
	uploader                  *s3manager.Uploader
	svc                       *s3.S3
	processUrlBeforeFetchFunc ProcessUrlBeforeFetchFuncType
	customizeFileNameFunc     CustomizeFileNameFuncType
	customizeFileExtFunc      CustomizeFileExtFuncType
}

func NewS3FileStore(bucket string) (*S3FileStore, error) {
	// AWS client session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-1"),
	})
	if err != nil {
		return nil, err
	}

	uploader := s3manager.NewUploader(sess)

	return &S3FileStore{
		bucket:                    bucket,
		uploader:                  uploader,
		svc:                       s3.New(session.Must(sess, err)),
		processUrlBeforeFetchFunc: func(s string) string { return s },
		customizeFileNameFunc:     nil,
		customizeFileExtFunc:      nil,
	}, nil
}

func (s *S3FileStore) SetProcessUrlBeforeFetchFunc(f ProcessUrlBeforeFetchFuncType) *S3FileStore {
	s.processUrlBeforeFetchFunc = f
	return s
}

func (s *S3FileStore) SetCustomizeFileNameFunc(f CustomizeFileNameFuncType) *S3FileStore {
	s.customizeFileNameFunc = f
	return s
}

func (s *S3FileStore) SetCustomizeFileExtFunc(f CustomizeFileExtFuncType) *S3FileStore {
	s.customizeFileExtFunc = f
	return s
}

// S3 key is the file name
func (s *S3FileStore) GenerateS3KeyFromUrl(url string) (key string, err error) {
	if s.customizeFileNameFunc != nil {
		key = s.customizeFileNameFunc(url)
	} else {
		key, err = utils.TextToMd5Hash(url)
	}

	if len(key) == 0 {
		err = errors.New("generate empty s3 key, invalid")
	}

	if s.customizeFileExtFunc != nil {
		key = key + "." + s.customizeFileExtFunc(url)
	} else {
		key = key + utils.GetUrlExtNameWithDot(url)
	}

	return key, err
}

// If url key existed, just return the existing key without update file
func (s *S3FileStore) FetchAndStore(url string) (key string, err error) {
	// Download file
	response, err := http.Get(s.processUrlBeforeFetchFunc(url))
	if err != nil {
		return "", err
	}

	key, err = s.GenerateS3KeyFromUrl(url)
	if err != nil {
		return "", err
	}

	if !s.IsKeyExisted(key) {
		// Upload the file to S3.
		_, err = s.uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
			Body:   response.Body,
		})
	}
	return key, err
}

func (s *S3FileStore) IsKeyExisted(key string) bool {
	_, err := s.svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String("bucket_name"),
		Key:    aws.String("object_key"),
	})
	if err != nil {
		return false
	}
	return true
}

func (s *S3FileStore) GetUrlFromKey(key string) string {
	return CouldFrontPrefix + key
}

func (s *S3FileStore) CleanUp() {
	// do nothing for s3
}
