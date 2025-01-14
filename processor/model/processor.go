package model

// LegalHoldProcessOptions contains all the options needed to process a legal hold
type LegalHoldProcessOptions struct {
	LegalHoldData   string
	OutputPath      string
	LegalHoldSecret string
}

// LegalHoldProcessResult contains the results of processing a legal hold
type LegalHoldProcessResult struct {
	LegalHolds []string // IDs of processed legal holds
	FilesCount int      // Number of files processed
}
