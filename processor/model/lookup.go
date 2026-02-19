package model

type (
	ChannelLookup        map[string]*LegalHoldChannel
	TeamLookup           map[string]*LegalHoldTeam
	TeamForChannelLookup map[string]*LegalHoldTeam
)

type FileLookup map[string]string // Key: FileID, Value: file path
