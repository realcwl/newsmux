package file_store

// Shared Func type for file stores
type ProcessUrlBeforeFetchFuncType func(string) string
type CustomizeFileNameFuncType func(string) string
type CustomizeFileExtFuncType func(string) string

type CollectedFileStore interface {
	FetchAndStore(url string) (key string, err error)
	GetUrlFromKey(key string) string
	CleanUp()
}
