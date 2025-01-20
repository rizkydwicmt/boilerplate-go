package cloudstorage

import (
	types "boilerplate-go/internal/common/type"
	"boilerplate-go/internal/pkg/helper"
	"boilerplate-go/internal/pkg/logger"
	"boilerplate-go/internal/service/cloud-storage/model"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"google.golang.org/api/option"
)

type Service struct {
	ctx    context.Context
	client *storage.Client
}

type IService interface {
	Upload(bucket, filename string, buffer []byte, size int, metaData map[string]string, mimeType string) (string, error)
	CheckBucket(bucket string) bool
	CreateBucket(name string) error
	Download(bucket, filename string) string
	Delete(bucket, filename string) error
	SetPublic(bucketName string)
	UploadSingle(file *types.BufferedFile, data model.UploadPost) (model.ResultDownload, error)
}

func NewService(ctx context.Context) (IService, error) {
	var client *storage.Client
	var err error

	credentials := map[string]string{
		"type":         "service_account",
		"project_id":   helper.GetEnv("CS_PROJECT_ID"),
		"client_email": helper.GetEnv("CS_ACCESS_KEY"),
		"private_key":  strings.ReplaceAll(helper.GetEnv("CS_SECRET_KEY"), "\\n", "\n"),
	}

	credentialsJSON, _ := json.Marshal(credentials)

	for retries := 0; retries < 5; retries++ {
		client, err = storage.NewClient(ctx, option.WithCredentialsJSON(credentialsJSON))

		if err == nil {
			break
		}

		logger.Warning.Println(fmt.Errorf("retry %d: failed to connect to storage Service: %w", retries+1, err), "Storage IService", "Connect", true)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		logger.Error.Println(fmt.Errorf("failed to connect to storage Service after retries: %w", err), "Storage IService", "Connect", true)
		return nil, err
	}

	return &Service{
		client: client,
		ctx:    ctx,
	}, nil
}

func (s *Service) Upload(bucket, filename string, buffer []byte, size int, metaData map[string]string, mimeType string) (string, error) {
	file := s.client.Bucket(bucket).Object(filename)
	stream := file.NewWriter(s.ctx)

	if _, err := stream.Write(buffer); err != nil {
		return "", err
	}
	if err := stream.Close(); err != nil {
		return "", err
	}

	if _, err := file.Update(s.ctx, storage.ObjectAttrsToUpdate{
		ContentType: mimeType,
		Metadata:    metaData,
	}); err != nil {
		return "", err
	}

	url := s.Download(bucket, filename)

	return url, nil
}

func (s *Service) CheckBucket(bucket string) bool {
	b, err := s.client.Bucket(bucket).Attrs(s.ctx)
	return err == nil && b.Name == bucket
}

func (s *Service) CreateBucket(name string) error {
	checkBucket := s.CheckBucket(name)
	if !checkBucket {
		if err := s.client.Bucket(name).Create(s.ctx, helper.GetEnv("CS_PROJECT_ID"), &storage.BucketAttrs{
			Location: "asia-southeast2",
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Download(bucket, filename string) string {
	url, err := s.client.Bucket(bucket).SignedURL(filename, &storage.SignedURLOptions{
		GoogleAccessID: helper.GetEnv("CS_ACCESS_KEY"),
		PrivateKey:     []byte(strings.ReplaceAll(helper.GetEnv("CS_SECRET_KEY"), "\\n", "\n")),
		Method:         "GET",
		Expires:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	if err != nil {
		return ""
	}
	return url
}

func (s *Service) Delete(bucket, filename string) error {
	file := s.client.Bucket(bucket).Object(filename)
	if err := file.Delete(s.ctx); err != nil {
		return err
	}
	return nil
}

func (s *Service) SetPublic(bucketName string) {
	bucket := s.client.Bucket(bucketName)
	if err := bucket.ACL().Set(s.ctx, storage.AllUsers, storage.RoleReader); err != nil {
		logger.Error.Println(fmt.Errorf("failed to set public access to bucket %s: %w", bucketName, err), "Cloud Storage IService", "SetPublic", true)
	}
}

func (s *Service) UploadSingle(file *types.BufferedFile, data model.UploadPost) (model.ResultDownload, error) {
	condition := data.Folder == "boilerplate-go" || data.Folder == "onx_dev"
	data.Folder = strings.ToLower(strings.TrimSpace(strings.Trim(data.Folder, "_")))

	if file.OriginalName == "" {
		nanoid, _ := gonanoid.Generate("1234567890abcdefghijklmnopqrstuvwxyz", 10)
		newName := nanoid
		file.OriginalName = newName
		if file.MimeType == "message/rfc822" {
			file.OriginalName += `.eml`
		}
	}

	baseBucket := data.Folder
	if helper.GetEnv("APP_ENV") != "production" && helper.GetEnv("APP_ENV") != "" {
		baseBucket = fmt.Sprintf("%s-%s", baseBucket, helper.GetEnv("APP_ENV"))
	}

	err := s.CreateBucket(baseBucket)
	if err != nil {
		logger.Error.Println(fmt.Errorf("failed to create bucket: %w", err), "Cloud Storage IService", "UploadSingle", true)
		return model.ResultDownload{}, err
	}

	ext := file.OriginalName[strings.LastIndex(file.OriginalName, "."):]
	tempFilename := time.Now().Format("2006-01-02-15-04-05")
	hashedFileName := fmt.Sprintf("%x", md5.Sum([]byte(tempFilename)))

	metaData := map[string]string{
		"Content-type": file.MimeType,
	}

	randNumber, _ := gonanoid.Generate("1234567890", 4)
	generateNewName := strings.Split(file.OriginalName, ".")[0] + "_" + randNumber + ext

	var filename string
	if file.OriginalName != "" {
		filename = generateNewName
	} else {
		filename = hashedFileName + ext
	}
	find := " "

	resFileName := strings.ReplaceAll(generateNewName, find, "_")

	filename = data.Directory + "/" + filename
	fileBuffer := file.Buffer
	fileSize := file.Size

	token, err := helper.EncryptAESCBC(fmt.Sprintf("%s:%s", baseBucket, filename))
	if err != nil {
		logger.Error.Println(err)
		return model.ResultDownload{}, err
	}
	newFilename := hashedFileName + ext
	if condition {
		newFilename = filename
	}
	url, err := s.Upload(baseBucket, filename, fileBuffer, fileSize, metaData, file.MimeType)
	if err != nil {
		logger.Error.Println(err)
		return model.ResultDownload{}, err
	}

	output := model.ResultDownload{
		URL:            url,
		OriginFileName: resFileName,
		MimeType:       file.MimeType,
		Size:           fileSize,
		FileName:       newFilename,
		Token:          token,
	}

	return output, nil
}
