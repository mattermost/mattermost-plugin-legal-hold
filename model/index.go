package model

// LegalHoldChannelMembership represents the membership of a channel by a user in the
// LegalHoldIndex.
type LegalHoldChannelMembership struct {
	ChannelID string
	StartTime int64
	EndTime   int64
}

// LegalHoldIndexUser represents the data about one user in the LegalHoldIndex.
type LegalHoldIndexUser struct {
	Username string
	Email    string
	Channels []LegalHoldChannelMembership
}

// LegalHoldIndex maps to the contents of the index.json file in a legal hold export.
// It contains various pieces of metadata to help with the programmatic and manual processing of
// the legal hold export.
type LegalHoldIndex map[string]LegalHoldIndexUser
