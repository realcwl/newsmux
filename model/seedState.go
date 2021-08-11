package model

type SeedState struct {
	UserSeedState *UserSeedState   `json:"userSeedState"`
	FeedSeedState []*FeedSeedState `json:"feedSeedState"`
}
