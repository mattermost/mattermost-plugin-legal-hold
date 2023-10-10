package model

// LegalHold represents one legal hold.
type LegalHold struct {
	ID                   string
	UserIDs              []string
	StartsAt             int64
	EndsAt               int64
	LastExecutionEndedAt int64
	ExecutionLength      int64
}

// LegalHoldCursor represents the state of a paginated LegalHold export query.
// It is based on the model.ComplianceCursor struct from Mattermost Server.
type LegalHoldCursor struct {
	LastPostCreateAt int64
	LastPostID       string
	BatchNumber      uint
	Completed        bool
}

// NewLegalHoldCursor creates a new LegalHoldCursor object with the provided startTime
// that is initialised and ready to use.
func NewLegalHoldCursor(startTime int64) LegalHoldCursor {
	return LegalHoldCursor{
		LastPostCreateAt: startTime,
		LastPostID:       "00000000000000000000000000",
		BatchNumber:      0,
		Completed:        false,
	}
}

// LegalHoldPost represents one post and its associated data as required for a legal hold record.
// It is based on the model.CompliancePost struct from Mattermost Server.
type LegalHoldPost struct {

	// From Team
	TeamName        string `csv:"TeamName"`
	TeamDisplayName string `csv:"TeamDisplayName"`

	// From Channel
	ChannelName        string `csv:"ChannelName"`
	ChannelDisplayName string `csv:"ChannelDisplayName"`
	ChannelType        string `csv:"ChannelType"`

	// From User
	UserUsername string `csv:"UserUsername"`
	UserEmail    string `csv:"UserEmail"`
	UserNickname string `csv:"UserNickname"`

	// From Post
	PostId         string `csv:"PostId"`
	PostCreateAt   int64  `csv:"PostCreateAt"`
	PostUpdateAt   int64  `csv:"PostUpdateAt"`
	PostDeleteAt   int64  `csv:"PostDeleteAt"`
	PostRootId     string `csv:"PostRootId"`
	PostOriginalId string `csv:"PostOriginalId"`
	PostMessage    string `csv:"PostMessage"`
	PostType       string `csv:"PostType"`
	PostProps      string `csv:"PostProps"`
	PostHashtags   string `csv:"PostHashtags"`
	PostFileIds    string `csv:"PostFileIds"`

	IsBot bool `csv:"IsBot"`
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

type LegalHoldChannelIndex map[string][]LegalHoldChannelMembership

type LegalHoldChannelMembership struct {
	ChannelID string
	StartTime int64
	EndTime   int64
}

func (*LegalHoldChannelIndex) Merge(new *LegalHoldChannelIndex) {
	// TODO: Implement me!
}
