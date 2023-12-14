package model

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMerge(t *testing.T) {
	oldIndex := LegalHoldIndex{
		"user1": {
			Username: "oldUser",
			Email:    "oldUser@example.com",
			Channels: []LegalHoldChannelMembership{
				{"channel1", 1000, 2000},
				{"channel3", 400, 4400},
			},
		},
		"user2": {
			Username: "user2",
			Email:    "user2@example.com",
			Channels: []LegalHoldChannelMembership{
				{"channel1", 1500, 2500},
				{"channel2", 3000, 4000},
			},
		},
	}

	newIndex := LegalHoldIndex{
		"user1": {
			Username: "newUser",
			Email:    "newUser@example.com",
			Channels: []LegalHoldChannelMembership{
				{"channel1", 1500, 2500},
				{"channel2", 3000, 4000},
			},
		},
		"user3": {
			Username: "user3",
			Email:    "user3@example.com",
			Channels: []LegalHoldChannelMembership{
				{"channel1", 1500, 2500},
				{"channel2", 3000, 4000},
			},
		},
	}

	expectedIndexAfterMerge := LegalHoldIndex{
		"user1": {
			Username: "newUser",
			Email:    "newUser@example.com",
			Channels: []LegalHoldChannelMembership{
				{"channel1", 1000, 2500},
				{"channel2", 3000, 4000},
				{"channel3", 400, 4400},
			},
		},
		"user2": {
			Username: "user2",
			Email:    "user2@example.com",
			Channels: []LegalHoldChannelMembership{
				{"channel1", 1500, 2500},
				{"channel2", 3000, 4000},
			},
		},
		"user3": {
			Username: "user3",
			Email:    "user3@example.com",
			Channels: []LegalHoldChannelMembership{
				{"channel1", 1500, 2500},
				{"channel2", 3000, 4000},
			},
		},
	}

	oldIndex.Merge(&newIndex)

	if !reflect.DeepEqual(oldIndex, expectedIndexAfterMerge) {
		t.Fail()
	}
}

func TestGetLegalHoldChannelMembership(t *testing.T) {
	type args struct {
		channelMemberships []LegalHoldChannelMembership
		channelID          string
	}

	tests := []struct {
		name                           string
		args                           args
		wantLegalHoldChannelMembership LegalHoldChannelMembership
		wantFound                      bool
	}{
		{
			name: "Test Case 1: Membership exists",
			args: args{
				channelMemberships: []LegalHoldChannelMembership{
					{ChannelID: "ch1"},
					{ChannelID: "ch2"},
				},
				channelID: "ch1",
			},
			wantLegalHoldChannelMembership: LegalHoldChannelMembership{ChannelID: "ch1"},
			wantFound:                      true,
		},
		{
			name: "Test Case 2: Membership does not exist",
			args: args{
				channelMemberships: []LegalHoldChannelMembership{
					{ChannelID: "ch1"},
					{ChannelID: "ch2"},
				},
				channelID: "ch3",
			},
			wantLegalHoldChannelMembership: LegalHoldChannelMembership{},
			wantFound:                      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLegalHoldChannelMembership, gotFound := getLegalHoldChannelMembership(tt.args.channelMemberships, tt.args.channelID)
			if gotLegalHoldChannelMembership != tt.wantLegalHoldChannelMembership {
				t.Errorf("getLegalHoldChannelMembership() got = %v, want %v", gotLegalHoldChannelMembership, tt.wantLegalHoldChannelMembership)
			}
			if gotFound != tt.wantFound {
				t.Errorf("getLegalHoldChannelMembership() got1 = %v, want %v", gotFound, tt.wantFound)
			}
		})
	}
}

func TestLegalHoldChannelMembership_Combine(t *testing.T) {
	// Initialize a new LegalHoldChannelMembership instance
	lhcm1 := LegalHoldChannelMembership{
		ChannelID: "testChannel1",
		StartTime: 10,
		EndTime:   20,
	}

	lhcm2 := LegalHoldChannelMembership{
		ChannelID: "testChannel2",
		StartTime: 5,
		EndTime:   25,
	}

	// Combine the two instances
	lhcmCombined := lhcm1.Combine(lhcm2)

	require.Equal(t, lhcm1.ChannelID, lhcmCombined.ChannelID)
	require.Equal(t, int64(5), lhcmCombined.StartTime)
	require.Equal(t, int64(25), lhcmCombined.EndTime)
}
