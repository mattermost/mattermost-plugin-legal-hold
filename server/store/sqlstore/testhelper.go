package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/mattermost/mattermost/server/public/shared/request"
	mmstore "github.com/mattermost/mattermost/server/v8/channels/store/sqlstore"
	"github.com/mattermost/mattermost/server/v8/platform/shared/filestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/utils"
)

const (
	dbDatabaseName = "mattermost_test"
)

type TestHelper struct {
	tearDowns []utils.TearDownFunc

	logger      *mlog.Logger
	mmStore     *mmstore.SqlStore
	Store       *SQLStore
	FileBackend filestore.FileBackend

	Team1    *model.Team
	Team2    *model.Team
	Channel1 *model.Channel
	Channel2 *model.Channel
	User1    *model.User
	User2    *model.User
}

func SetupHelper(t *testing.T) *TestHelper {
	th := &TestHelper{
		tearDowns: make([]utils.TearDownFunc, 0),
	}

	// TODO: uncomment below
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour*30)
	defer cancel()

	dbDriverName := os.Getenv("TEST_DB_DRIVER_NAME")
	if len(dbDriverName) == 0 {
		dbDriverName = "postgres"
	}

	if dbDriverName != "postgres" {
		t.Fatalf("invalid TEST_DB_DRIVER_NAME environment variable: %s", dbDriverName)
	}

	// create and initialize a new database
	connStr, dbTearDown, err := utils.CreateTestDB(ctx, dbDriverName, dbDatabaseName)
	if err != nil {
		require.NoError(t, err, "cannot instantiate test database")
	}
	th.tearDowns = append(th.tearDowns, dbTearDown)

	settings := model.SqlSettings{}
	settings.SetDefaults(false)
	*settings.DataSource = connStr
	*settings.DriverName = dbDriverName

	t.Log("using database connection string: ", connStr)

	logger, err := mlog.NewLogger()
	require.NoError(t, err)
	th.logger = logger
	th.tearDowns = append(th.tearDowns, func(context.Context) error {
		return logger.Shutdown()
	})

	th.mmStore, err = mmstore.New(settings, logger, nil)
	require.NoError(t, err, "cannot instantiate test database")
	th.tearDowns = append(th.tearDowns, func(context.Context) error {
		th.mmStore.Close()
		return nil
	})

	store, err := New(storeWrapper{th.mmStore}, &testLogger{t})
	require.NoError(t, err, "could not create store")
	th.Store = store

	// create a new minio instance
	connStr, dbTearDown, err = utils.CreateMinio(ctx)
	if err != nil {
		require.NoError(t, err, "cannot instantiate test minio")
	}
	th.tearDowns = append(th.tearDowns, dbTearDown)

	fileBackendSettings := utils.GetBackendSettings(connStr)
	fileBackend, err := filestore.NewFileBackend(fileBackendSettings)
	require.NoError(t, err)
	require.NoError(t, fileBackend.TestConnection())
	th.FileBackend = fileBackend

	return th
}

func (th *TestHelper) SetupBasic(t *testing.T) *TestHelper {
	// create some teams
	teams, err := th.CreateTeams(2, "test-team")
	require.NoError(t, err, "could not create teams")
	th.Team1 = teams[0]
	th.Team2 = teams[1]

	// create some users
	users, err := th.CreateUsers(2, "test.user")
	require.NoError(t, err)
	th.User1 = users[0]
	th.User2 = users[1]

	// create some channels
	channels, err := th.CreateChannels(2, "test-channel", th.User1.Id, th.Team1.Id)
	require.NoError(t, err, "could not create channels")
	th.Channel1 = channels[0]
	th.Channel2 = channels[1]

	return th
}

func (th *TestHelper) TearDown(t *testing.T) {
	ctx := context.Background()
	// Tear down in reverse order, so each resource closes before the one it depends on.
	for _, tearDown := range slices.Backward(th.tearDowns) {
		assert.NoError(t, tearDown(ctx))
	}
}

func (th *TestHelper) CreateTeams(num int, namePrefix string) ([]*model.Team, error) {
	var teams []*model.Team
	for i := range num {
		team := &model.Team{
			Name:        fmt.Sprintf("%s-%d", namePrefix, i),
			DisplayName: fmt.Sprintf("%s-%d", namePrefix, i),
			Type:        model.TeamOpen,
		}
		team, err := th.mmStore.Team().Save(team)
		if err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}
	return teams, nil
}

func (th *TestHelper) CreateOpenChannel(name string, userID string, teamID string) (*model.Channel, error) {
	return th.CreateChannel(name, userID, teamID, model.ChannelTypeOpen)
}

func (th *TestHelper) CreateChannel(name, userID, teamID string, channelType model.ChannelType) (*model.Channel, error) {
	rctx := request.EmptyContext(th.logger)

	channel := &model.Channel{
		Name:        name,
		DisplayName: name,
		Type:        channelType,
		CreatorId:   userID,
		TeamId:      teamID,
	}
	return th.mmStore.Channel().Save(rctx, channel, 1024)
}

func (th *TestHelper) CreateChannels(num int, namePrefix string, userID string, teamID string) ([]*model.Channel, error) {
	rctx := request.EmptyContext(th.logger)
	var channels []*model.Channel
	for i := range num {
		channel := &model.Channel{
			Name:        fmt.Sprintf("%s-%d", namePrefix, i),
			DisplayName: fmt.Sprintf("%s-%d", namePrefix, i),
			Type:        model.ChannelTypeOpen,
			CreatorId:   userID,
			TeamId:      teamID,
		}
		channel, err := th.mmStore.Channel().Save(rctx, channel, 1024)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	return channels, nil
}

func (th *TestHelper) CreateChannelsWithChannelMemberHistory(num int, namePrefix string, userID string, teamID string) ([]*model.Channel, error) {
	rctx := request.EmptyContext(th.logger)

	var channels []*model.Channel
	for i := range num {
		channel := &model.Channel{
			Name:        fmt.Sprintf("%s-%d", namePrefix, i),
			DisplayName: fmt.Sprintf("%s-%d", namePrefix, i),
			Type:        model.ChannelTypeOpen,
			CreatorId:   userID,
			TeamId:      teamID,
		}
		channel, err := th.mmStore.Channel().Save(rctx, channel, 1024)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)

		err = th.mmStore.ChannelMemberHistory().LogJoinEvent(userID, channel.Id, model.GetMillis())
		if err != nil {
			return nil, err
		}
	}
	return channels, nil
}

func (th *TestHelper) CreateDirectMessageChannel(user1 *model.User, user2 *model.User) (*model.Channel, error) {
	rctx := request.EmptyContext(th.logger)

	return th.mmStore.Channel().CreateDirectChannel(rctx, user1, user2)
}

func (th *TestHelper) CreateUsers(num int, namePrefix string) ([]*model.User, error) {
	rctx := request.EmptyContext(th.logger)

	var users []*model.User
	for i := range num {
		user := &model.User{
			Username: fmt.Sprintf("%s-%d", namePrefix, i),
			Password: namePrefix,
			Email:    fmt.Sprintf("%s@example.com", model.NewId()),
		}
		user, err := th.mmStore.User().Save(rctx, user)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (th *TestHelper) CreatePosts(num int, userID string, channelID string) ([]*model.Post, error) {
	rctx := request.EmptyContext(th.logger)

	var posts []*model.Post
	for i := range num {
		post := &model.Post{
			UserId:    userID,
			ChannelId: channelID,
			Type:      model.PostTypeDefault,
			Message:   fmt.Sprintf("test post %d of %d", i, num),
		}
		post, err := th.mmStore.Post().Save(rctx, post)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, nil
}

func (th *TestHelper) CreatePostsWithAttachments(num int, userID string, channelID string) ([]*model.Post, error) {
	rctx := request.EmptyContext(th.logger)
	var posts []*model.Post
	for i := range num {
		text := "This is a test uploaded file."
		reader := strings.NewReader(text)
		size, err := th.FileBackend.WriteFile(reader, "data/file_upload_test.txt")
		if err != nil {
			return nil, err
		}

		fileInfo := &model.FileInfo{
			Id:        model.NewId(),
			CreateAt:  model.GetMillis(),
			UpdateAt:  model.GetMillis(),
			CreatorId: userID,
			Name:      "file_upload_test.txt",
			Path:      "data/file_upload_test.txt",
			MimeType:  "text/plain",
			Size:      size,
		}

		fileInfo, err = th.mmStore.FileInfo().Save(rctx, fileInfo)
		if err != nil {
			return nil, err
		}

		post := &model.Post{
			UserId:    userID,
			ChannelId: channelID,
			Type:      model.PostTypeDefault,
			Message:   fmt.Sprintf("test post %d of %d", i, num),
			FileIds:   []string{fileInfo.Id},
		}
		post, err = th.mmStore.Post().Save(rctx, post)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, nil
}

func (th *TestHelper) CreateReactions(posts []*model.Post, userID string) ([]*model.Reaction, error) {
	var reactions []*model.Reaction
	for _, post := range posts {
		reaction := &model.Reaction{
			PostId:    post.Id,
			UserId:    userID,
			EmojiName: "shrug",
			ChannelId: post.ChannelId,
		}
		reaction, err := th.mmStore.Reaction().Save(reaction)
		if err != nil {
			return nil, err
		}
		reactions = append(reactions, reaction)
	}
	return reactions, nil
}

// storeWrapper is a wrapper for MainHelper that implements SQLStoreSource interface.
type storeWrapper struct {
	mmStore *mmstore.SqlStore
}

func (sw storeWrapper) GetMasterDB() (*sql.DB, error) {
	return sw.mmStore.GetInternalMasterDB(), nil
}

func (sw storeWrapper) GetReplicaDB() (*sql.DB, error) {
	// For this test helper, just return the master DB even when a replica has been asked for.
	return sw.mmStore.GetInternalMasterDB(), nil
}

func (sw storeWrapper) DriverName() string {
	return sw.mmStore.DriverName()
}

type testLogger struct {
	tb testing.TB
}

// Error logs an error message, optionally structured with alternating key, value parameters.
func (l *testLogger) Error(message string, keyValuePairs ...any) {
	l.log("error", message, keyValuePairs...)
}

// Warn logs an error message, optionally structured with alternating key, value parameters.
func (l *testLogger) Warn(message string, keyValuePairs ...any) {
	l.log("warn", message, keyValuePairs...)
}

// Info logs an error message, optionally structured with alternating key, value parameters.
func (l *testLogger) Info(message string, keyValuePairs ...any) {
	l.log("info", message, keyValuePairs...)
}

// Debug logs an error message, optionally structured with alternating key, value parameters.
func (l *testLogger) Debug(message string, keyValuePairs ...any) {
	l.log("debug", message, keyValuePairs...)
}

func (l *testLogger) log(level string, message string, keyValuePairs ...any) {
	var args strings.Builder

	if len(keyValuePairs) > 0 && len(keyValuePairs)%2 != 0 {
		keyValuePairs = keyValuePairs[:len(keyValuePairs)-1]
	}

	for i := 0; i < len(keyValuePairs); i += 2 {
		fmt.Fprintf(&args, "%v:%v  ", keyValuePairs[i], keyValuePairs[i+1])
	}

	l.tb.Logf("level=%s  message=%s  %s", level, message, args.String())
}
