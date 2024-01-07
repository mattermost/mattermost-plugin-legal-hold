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
