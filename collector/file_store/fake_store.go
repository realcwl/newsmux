package file_store

type FakeFileStore struct{}

func (*FakeFileStore) FetchAndStore(url string, fileName string) (key string, err error) {
	return url + fileName, nil
}
func (*FakeFileStore) GetUrlFromKey(key string) string {
	return key
}

func (*FakeFileStore) CleanUp() {}
