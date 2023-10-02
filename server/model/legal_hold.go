package model

// LegalHoldCursor represents the state of a paginated LegalHold export query.
// It is based on the model.ComplianceCursor struct from Mattermost Server.
type LegalHoldCursor struct {
	LastChannelsQueryPostCreateAt       int64
	LastChannelsQueryPostID             string
	ChannelsQueryCompleted              bool
	LastDirectMessagesQueryPostCreateAt int64
	LastDirectMessagesQueryPostID       string
	DirectMessagesQueryCompleted        bool
}

func NewLegalHoldCursor(startTime int64) LegalHoldCursor {
	return LegalHoldCursor{
		LastChannelsQueryPostCreateAt:       startTime,
		LastChannelsQueryPostID:             "00000000000000000000000000",
		LastDirectMessagesQueryPostCreateAt: startTime,
		LastDirectMessagesQueryPostID:       "00000000000000000000000000",
	}
}

// LegalHoldPost represents one post and its associated data as required for a legal hold record.
// It is based on the model.CompliancePost struct from Mattermost Server.
type LegalHoldPost struct {

	// From Team
	TeamName        string
	TeamDisplayName string

	// From Channel
	ChannelName        string
	ChannelDisplayName string
	ChannelType        string

	// From User
	UserUsername string
	UserEmail    string
	UserNickname string

	// From Post
	PostId         string
	PostCreateAt   int64
	PostUpdateAt   int64
	PostDeleteAt   int64
	PostRootId     string
	PostOriginalId string
	PostMessage    string
	PostType       string
	PostProps      string
	PostHashtags   string
	PostFileIds    string

	IsBot bool
}

// LegalHoldPostHeader returns the headers for a tabulated representation of LegalHoldPost structs.
// It is based on the model.CompliancePostHeader function from Mattermost Server.
func LegalHoldPostHeader() []string {
	return []string{
		"TeamName",
		"TeamDisplayName",

		"ChannelName",
		"ChannelDisplayName",
		"ChannelType",

		"UserUsername",
		"UserEmail",
		"UserNickname",
		"UserType",

		"PostId",
		"PostCreateAt",
		"PostUpdateAt",
		"PostDeleteAt",
		"PostRootId",
		"PostOriginalId",
		"PostMessage",
		"PostType",
		"PostProps",
		"PostHashtags",
		"PostFileIds",
	}
}
