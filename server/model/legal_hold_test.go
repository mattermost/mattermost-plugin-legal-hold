package model

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	mattermostModel "github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModel_LegalHold_DeepCopy(t *testing.T) {
	testCases := []struct {
		name string
		lh   *LegalHold
	}{
		{
			name: "Nil Legal Hold",
			lh:   nil,
		},
		{
			name: "Empty Legal Hold",
			lh:   &LegalHold{},
		},
		{
			name: "Legal Hold with Fields",
			lh: &LegalHold{
				ID:                   "Test ID",
				Name:                 "Test Name",
				DisplayName:          "Test Display Name",
				CreateAt:             12345,
				UpdateAt:             12355,
				UserIDs:              []string{"UserID1", "UserID2"},
				StartsAt:             12360,
				EndsAt:               12370,
				LastExecutionEndedAt: 12365,
				ExecutionLength:      30,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.lh.DeepCopy()
			if tc.lh != nil {
				assert.Equal(t, *tc.lh, result)
				result.ID = "Changed ID"
				assert.NotEqual(t, *tc.lh, result)

				if len(result.UserIDs) > 0 {
					result.UserIDs[0] = "Changed ID"
					assert.NotEqual(t, tc.lh.UserIDs[0], result.UserIDs[0])
				}
			} else {
				emptyLH := LegalHold{}
				assert.Equal(t, emptyLH, result)
			}
		})
	}
}

func TestModel_LegalHold_IsValidForCreate(t *testing.T) {
	tests := []struct {
		name    string
		lh      *LegalHold
		wantErr bool
	}{
		{
			name: "Valid",
			lh: &LegalHold{
				ID:                    mattermostModel.NewId(),
				Name:                  "legalhold1",
				DisplayName:           "Test Legal Hold",
				UserIDs:               []string{mattermostModel.NewId(), mattermostModel.NewId()},
				StartsAt:              10,
				EndsAt:                0,
				IncludePublicChannels: false,
			},
			wantErr: false,
		},
		{
			name: "Invalid ID",
			lh: &LegalHold{
				ID:          "test ID",
				Name:        "legalhold1",
				DisplayName: "Invalid ID Test",
				UserIDs:     []string{mattermostModel.NewId(), mattermostModel.NewId()},
				StartsAt:    20,
				EndsAt:      0,
			},
			wantErr: true,
		},
		{
			name: "Invalid Name",
			lh: &LegalHold{
				ID:          mattermostModel.NewId(),
				Name:        "Invalid Name#",
				DisplayName: "Invalid Name Test",
				UserIDs:     []string{mattermostModel.NewId(), mattermostModel.NewId()},
				StartsAt:    30,
				EndsAt:      0,
			},
			wantErr: true,
		},
		{
			name: "Name length less than 2",
			lh: &LegalHold{
				ID:          mattermostModel.NewId(),
				Name:        "l",
				DisplayName: "Short Name Test",
				UserIDs:     []string{mattermostModel.NewId(), mattermostModel.NewId()},
				StartsAt:    40,
				EndsAt:      0,
			},
			wantErr: true,
		},
		{
			name: "Name length greater than 64",
			lh: &LegalHold{
				ID:          mattermostModel.NewId(),
				Name:        "longtestnamelongtestnamelongtestnamelongtestnamelongtestnamelongtestnamelongtestnamelongtestnamelongtestnamelongtestnamelongtestnamelongtestname",
				DisplayName: "Long Name Test",
				UserIDs:     []string{mattermostModel.NewId(), mattermostModel.NewId()},
				StartsAt:    50,
				EndsAt:      0,
			},
			wantErr: true,
		},
		{
			name: "DisplayName length less than 2",
			lh: &LegalHold{
				ID:          mattermostModel.NewId(),
				Name:        "legalhold1",
				DisplayName: "D",
				UserIDs:     []string{mattermostModel.NewId(), mattermostModel.NewId()},
				StartsAt:    60,
				EndsAt:      0,
			},
			wantErr: true,
		},
		{
			name: "DisplayName length greater than 64",
			lh: &LegalHold{
				ID:          mattermostModel.NewId(),
				Name:        "legalhold1",
				DisplayName: "LongDisplayNameTestLongDisplayNameTestLongDisplayNameTestLongDisplayNameTest123",
				UserIDs:     []string{mattermostModel.NewId(), mattermostModel.NewId()},
				StartsAt:    50,
				EndsAt:      0,
			},
			wantErr: true,
		},
		{
			name: "No UserID",
			lh: &LegalHold{
				ID:          mattermostModel.NewId(),
				Name:        "legalhold1",
				DisplayName: "No UserID Test",
				UserIDs:     []string{},
				StartsAt:    60,
				EndsAt:      0,
			},
			wantErr: true,
		},
		{
			name: "Invalid UserID",
			lh: &LegalHold{
				ID:          mattermostModel.NewId(),
				Name:        "legalhold1",
				DisplayName: "Invalid UserID Test",
				UserIDs:     []string{mattermostModel.NewId(), "invalid user"},
				StartsAt:    70,
				EndsAt:      0,
			},
			wantErr: true,
		},
		{
			name: "Invalid StartsAt",
			lh: &LegalHold{
				ID:          mattermostModel.NewId(),
				Name:        "legalhold1",
				DisplayName: "Invalid StartsAt Test",
				UserIDs:     []string{mattermostModel.NewId(), mattermostModel.NewId()},
				StartsAt:    0,
				EndsAt:      0,
			},
			wantErr: true,
		},
		{
			name: "Invalid EndsAt",
			lh: &LegalHold{
				ID:          mattermostModel.NewId(),
				Name:        "legalhold1",
				DisplayName: "Invalid EndsAt Test",
				UserIDs:     []string{mattermostModel.NewId(), mattermostModel.NewId()},
				StartsAt:    80,
				EndsAt:      -1,
			},
			wantErr: true,
		},
		{
			name: "EndsAt before StartsAt",
			lh: &LegalHold{
				ID:          mattermostModel.NewId(),
				Name:        "legalhold1",
				DisplayName: "EndsAt before StartsAt Test",
				UserIDs:     []string{mattermostModel.NewId(), mattermostModel.NewId()},
				StartsAt:    90,
				EndsAt:      80,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.lh.IsValidForCreate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestModel_LegalHold_NeedsExecuting(t *testing.T) {
	tests := []struct {
		name string
		now  int64
		lh   LegalHold
		want bool
	}{
		{
			name: "Starts in the future",
			now:  10,
			lh: LegalHold{
				LastExecutionEndedAt: 0,
				StartsAt:             20,
				ExecutionLength:      30,
				EndsAt:               40,
			},
			want: false,
		},
		{
			name: "Ends in the past, not yet finished",
			now:  50,
			lh: LegalHold{
				LastExecutionEndedAt: 20,
				StartsAt:             10,
				ExecutionLength:      20,
				EndsAt:               40,
			},
			want: true,
		},
		{
			name: "Ends in the past, not yet finished, short final run",
			now:  50,
			lh: LegalHold{
				LastExecutionEndedAt: 20,
				StartsAt:             10,
				ExecutionLength:      20,
				EndsAt:               30,
			},
			want: true,
		},
		{
			name: "Ends in the future, execution would end in future",
			now:  10,
			lh: LegalHold{
				LastExecutionEndedAt: 0,
				StartsAt:             5,
				ExecutionLength:      10,
				EndsAt:               20,
			},
			want: false,
		},
		{
			name: "Ends in the future, execution would end in past",
			now:  30,
			lh: LegalHold{
				LastExecutionEndedAt: 15,
				StartsAt:             5,
				ExecutionLength:      10,
				EndsAt:               50,
			},
			want: true,
		},
		{
			name: "No End time, execution would end in future",
			now:  40,
			lh: LegalHold{
				LastExecutionEndedAt: 30,
				StartsAt:             30,
				ExecutionLength:      20,
				EndsAt:               0,
			},
			want: false,
		},
		{
			name: "No End time, execution would end in past",
			now:  60,
			lh: LegalHold{
				LastExecutionEndedAt: 30,
				StartsAt:             30,
				ExecutionLength:      20,
				EndsAt:               0,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.lh.NeedsExecuting(tt.now)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestModel_LegalHold_NextExecutionStartTime(t *testing.T) {
	lh := LegalHold{
		StartsAt:             10,
		LastExecutionEndedAt: 20,
	}
	assert.Equal(t, int64(20), lh.NextExecutionStartTime())

	lh = LegalHold{
		StartsAt:             20,
		LastExecutionEndedAt: 10,
	}
	assert.Equal(t, int64(20), lh.NextExecutionStartTime())

	lh = LegalHold{
		StartsAt:             10,
		LastExecutionEndedAt: 10,
	}
	assert.Equal(t, int64(10), lh.NextExecutionStartTime())
}

func TestModel_LegalHold_NextExecutionEndTime(t *testing.T) {
	lh := &LegalHold{
		StartsAt:             1,
		LastExecutionEndedAt: 10,
		ExecutionLength:      5,
		EndsAt:               20,
	}

	t.Run("NextExecutionEndTime before EndsAt", func(t *testing.T) {
		expected := int64(15)
		assert.Equal(t, expected, lh.NextExecutionEndTime())
	})

	lh.EndsAt = 12
	t.Run("NextExecutionEndTime equals EndsAt", func(t *testing.T) {
		expected := int64(12)
		assert.Equal(t, expected, lh.NextExecutionEndTime())
	})

	lh.EndsAt = 0
	t.Run("NextExecutionEndTime when no EndsAt defined", func(t *testing.T) {
		expected := int64(15)
		assert.Equal(t, expected, lh.NextExecutionEndTime())
	})
}

func TestModel_LegalHold_IsFinished(t *testing.T) {
	tests := []struct {
		name               string
		lastExecutionEnded int64
		endsAt             int64
		want               bool
	}{
		{
			"If LastExecutionEndedAt is after EndsAt, It is finished",
			time.Date(2023, time.April, 13, 12, 30, 0, 0, time.UTC).UnixMilli(),
			time.Date(2023, time.April, 13, 12, 0, 0, 0, time.UTC).UnixMilli(),
			true,
		},
		{
			"If LastExecutionEndedAt is before EndsAt, It is not finished",
			time.Date(2023, time.April, 13, 11, 30, 0, 0, time.UTC).UnixMilli(),
			time.Date(2023, time.April, 13, 12, 0, 0, 0, time.UTC).UnixMilli(),
			false,
		},
		{
			"If LastExecutionEndedAt is exactly on EndsAt, It is finished",
			time.Date(2023, time.April, 13, 12, 0, 0, 0, time.UTC).UnixMilli(),
			time.Date(2023, time.April, 13, 12, 0, 0, 0, time.UTC).UnixMilli(),
			true,
		},
		{
			"If EndsAt is zero then it is never finished.",
			time.Date(2023, time.April, 13, 12, 0, 0, 0, time.UTC).UnixMilli(),
			int64(0),
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lh := &LegalHold{
				LastExecutionEndedAt: tc.lastExecutionEnded,
				EndsAt:               tc.endsAt,
			}

			got := lh.IsFinished()
			if got != tc.want {
				t.Errorf("IsFinished() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestModel_LegalHold_BasePath(t *testing.T) {
	cases := []struct {
		lh       *LegalHold
		expected string
	}{
		{&LegalHold{Name: "testhold", ID: "1"}, "legal_hold/testhold_1"},
		{&LegalHold{Name: "anotherhold", ID: "2"}, "legal_hold/anotherhold_2"},
	}

	for _, tc := range cases {
		result := tc.lh.BasePath()
		if result != tc.expected {
			t.Errorf("BasePath() = %s; expected %s", result, tc.expected)
		}
	}
}

func TestModel_UpdateLegalHold_IsValid(t *testing.T) {
	testCases := []struct {
		name     string
		ulh      UpdateLegalHold
		expected string
	}{
		{
			name: "Valid",
			ulh: UpdateLegalHold{
				ID:                    model.NewId(),
				DisplayName:           "TestName",
				UserIDs:               []string{model.NewId()},
				EndsAt:                0,
				IncludePublicChannels: false,
			},
			expected: "",
		},
		{
			name: "InvalidId",
			ulh: UpdateLegalHold{
				ID:          "abc",
				DisplayName: "TestName",
				UserIDs:     []string{model.NewId()},
				EndsAt:      0,
			},
			expected: "LegalHold ID is not valid: abc",
		},
		{
			name: "InvalidDisplayName",
			ulh: UpdateLegalHold{
				ID:          model.NewId(),
				DisplayName: "T",
				UserIDs:     []string{model.NewId()},
				EndsAt:      0,
			},
			expected: "LegalHold display name must be between 2 and 64 characters in length",
		},
		{
			name: "EmptyUserIDs",
			ulh: UpdateLegalHold{
				ID:          model.NewId(),
				DisplayName: "TestName",
				UserIDs:     []string{},
				EndsAt:      0,
			},
			expected: "LegalHold must include at least 1 user",
		},
		{
			name: "InvalidUserIDs",
			ulh: UpdateLegalHold{
				ID:          model.NewId(),
				DisplayName: "TestName",
				UserIDs:     []string{"abc"},
				EndsAt:      0,
			},
			expected: "LegalHold users must have valid IDs",
		},
		{
			name: "NegativeEndsAt",
			ulh: UpdateLegalHold{
				ID:                    model.NewId(),
				DisplayName:           "TestName",
				UserIDs:               []string{model.NewId()},
				IncludePublicChannels: false,
				EndsAt:                -1,
			},
			expected: "LegalHold must end at a valid time or zero",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.ulh.IsValid()
			if err != nil {
				if err.Error() != testCase.expected {
					t.Errorf("expected: %s, got: %s", testCase.expected, err.Error())
				}
			} else if testCase.expected != "" {
				t.Errorf("expected: %s, got: nil", testCase.expected)
			}
		})
	}
}
