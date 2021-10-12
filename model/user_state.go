package model

type UserState struct {
	User *User `json:"user"`
}

type UserStateInput struct {
	UserID string `json:"userId"`
}
