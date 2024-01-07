package model

type User struct {
	ID       string
	Username string
	Email    string
}

func NewUserFromIDAndIndex(userID string, index LegalHoldIndexUser) User {
	return User{
		ID:       userID,
		Username: index.Username,
		Email:    index.Email,
	}
}

type UserWithChannels struct {
	ID       string
	Username string
	Email    string
	Channels []string
}

func NewUserWithChannelsFromIDAndIndex(userID string, index LegalHoldIndexUser) UserWithChannels {
	return UserWithChannels{
		ID:       userID,
		Username: index.Username,
		Email:    index.Email,
		Channels: []string{},
	}
}
