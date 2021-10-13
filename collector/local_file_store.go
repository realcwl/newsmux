package collector

import (
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/Luismorlan/newsmux/utils"
)

type LocalFileStore struct {
	processUrlBeforeFetchFunc ProcessUrlBeforeFetchFuncType
	customizeFileNameFunc     CustomizeFileNameFuncType
	customizeFileExtFunc      CustomizeFileExtFuncType
}

func NewLocalFileStore(bucket string) (*LocalFileStore, error) {
	return &LocalFileStore{
		processUrlBeforeFetchFunc: func(s string) string { return s },
		customizeFileNameFunc:     func(s string) string { return s },
		customizeFileExtFunc:      func(s string) string { return s },
	}, nil
}

func (s *LocalFileStore) SetProcessUrlBeforeFetchFunc(f ProcessUrlBeforeFetchFuncType) *LocalFileStore {
	s.processUrlBeforeFetchFunc = f
	return s
}

func (s *LocalFileStore) SetCustomizeFileNameFunc(f CustomizeFileNameFuncType) *LocalFileStore {
	s.customizeFileNameFunc = f
	return s
}

func (s *LocalFileStore) SetCustomizeFileExtFunc(f CustomizeFileExtFuncType) *LocalFileStore {
	s.customizeFileExtFunc = f
	return s
}

func (s *LocalFileStore) GenerateFileNameFromUrl(url string) (key string, err error) {
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
	}

	return key, err
}

func (s *LocalFileStore) FetchAndStore(url string) error {
	// Download file
	response, err := http.Get(s.processUrlBeforeFetchFunc(url))
	if err != nil {
		return err
	}
	defer response.Body.Close()

	fileName, err := s.GenerateFileNameFromUrl(url)

	//open a file for writing
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	// Use io.Copy to just dump the response body to the file. This supports huge files
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	return err
}
