package model

type ChannelLookup map[string]*LegalHoldChannel
type TeamLookup map[string]*LegalHoldTeam
type TeamForChannelLookup map[string]*LegalHoldTeam

type FileLookup map[string]string // Key: FileID, Value: file path
