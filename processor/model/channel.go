package model

import "math"

type Channel struct {
	Path       string
	ID         string
	LowerBound int64
	UpperBound int64
}

func NewChannel(path string, id string) Channel {
	return Channel{
		Path:       path,
		ID:         id,
		LowerBound: 0,
		UpperBound: math.MaxInt64,
	}
}

func NewChannelWithBounds(path string, id string, lowerBound int64, upperBound int64) Channel {
	return Channel{
		Path:       path,
		ID:         id,
		LowerBound: lowerBound,
		UpperBound: upperBound,
	}
}
