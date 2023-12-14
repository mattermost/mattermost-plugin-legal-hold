package model

import (
	"strings"
	"testing"
	"time"

	mattermostModel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/stretchr/testify/assert"
)

func TestModel_LegalHold_IsValidForCreate(t *testing.T) {
	tests := []struct {
		name string
		lh   LegalHold
		err  bool
	}{
		{
			name: "invalid ID",
			lh: LegalHold{
				ID: "invalid",
			},
			err: true,
		},
		{
			name: "invalid Name",
			lh: LegalHold{
				ID:   mattermostModel.NewId(),
				Name: "a-s-d f",
			},
			err: true,
		},
		{
			name: "name too short",
			lh: LegalHold{
				ID:   mattermostModel.NewId(),
				Name: "a",
			},
			err: true,
		},
		{
			name: "name too long",
			lh: LegalHold{
				ID:   mattermostModel.NewId(),
				Name: strings.Repeat("a", 65),
			},
			err: true,
		},
		{
			name: "all valid",
			lh: LegalHold{
				ID:   mattermostModel.NewId(),
				Name: "asdf-foo1",
			},
			err: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.lh.IsValidForCreate()
			if (err != nil) != tt.err {
				t.Errorf("LegalHold.IsValidForCreate() error = %v, wantErr %v", err, tt.err)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.lh.NeedsExecuting(tt.now)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsFinished(t *testing.T) {
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

func TestBasePath(t *testing.T) {
	cases := []struct {
		lh       *LegalHold
		expected string
	}{
		{&LegalHold{Name: "testhold", ID: "1"}, "legal_hold/testhold_(1)"},
		{&LegalHold{Name: "anotherhold", ID: "2"}, "legal_hold/anotherhold_(2)"},
	}

	for _, tc := range cases {
		result := tc.lh.BasePath()
		if result != tc.expected {
			t.Errorf("BasePath() = %s; expected %s", result, tc.expected)
		}
	}
}
