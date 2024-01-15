package model

// LegalHoldChannelMembership represents the membership of a channel by a user in the
// LegalHoldIndexUsers.
type LegalHoldChannelMembership struct {
	ChannelID string `json:"channel_id"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
}

// LegalHoldIndexUser represents the data about one user in the LegalHoldIndexUsers.
type LegalHoldIndexUser struct {
	Username string                       `json:"username"`
	Email    string                       `json:"email"`
	Channels []LegalHoldChannelMembership `json:"channels"`
}

// LegalHoldIndexUsers maps to the contents of the index.json file in a legal hold export.
// It contains various pieces of metadata to help with the programmatic and manual processing of
// the legal hold export.
type LegalHoldIndexUsers map[string]LegalHoldIndexUser

type LegalHoldIndexDetails struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	DisplayName          string `json:"display_name"`
	StartsAt             int64  `json:"starts_at"`
	LastExecutionEndedAt int64  `json:"last_execution_ended_at"`
}

type LegalHoldIndex struct {
	Users     LegalHoldIndexUsers   `json:"users"`
	LegalHold LegalHoldIndexDetails `json:"legal_hold"`
	Teams     []*LegalHoldTeam      `json:"teams"`
}

type LegalHoldTeam struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	DisplayName string              `json:"display_name"`
	Channels    []*LegalHoldChannel `json:"channels"`
}

type LegalHoldChannel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}
