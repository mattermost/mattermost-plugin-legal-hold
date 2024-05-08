package sqlstore

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/shared/filestore"
	"github.com/mattermost/mattermost-server/v6/store/storetest"
	"github.com/mattermost/mattermost-server/v6/testlib"
)

var onceStartDocker sync.Once

type TestHelper struct {
	mainHelper *testlib.MainHelper
	restoreEnv map[string]string

	Store       *SQLStore
	FileBackend filestore.FileBackend

	Team1    *model.Team
	Team2    *model.Team
	Channel1 *model.Channel
	Channel2 *model.Channel
	User1    *model.User
	User2    *model.User
}

func getServerPath(t *testing.T) string {
	out, err := exec.Command("go", "list", "-m", "-f", "'{{.Dir}}'", "github.com/mattermost/mattermost-server/v6").Output()
	require.NoError(t, err, "cannot get mod cache path for server package")
	return strings.Trim(strings.TrimSpace(string(out)), "'")
}

func SetupHelper(t *testing.T) *TestHelper {
	var options = testlib.HelperOptions{
		EnableStore: true,
	}

	// testlib needs to access files in the server package, so here we set the
	// MM_SERVER_PATH env var to point to the server package in mod cache.
	restoreEnv := make(map[string]string)
	serverPath := getServerPath(t)
	if serverPath != "" {
		oldPath := os.Getenv("MM_SERVER_PATH")
		err := os.Setenv("MM_SERVER_PATH", serverPath)
		require.NoError(t, err, "cannot set env MM_SERVER_PATH var")
		restoreEnv["MM_SERVER_PATH"] = oldPath
	}

	t.Logf("serverPath=%s", serverPath)
	t.Logf("MM_SERVER_PATH=%s", os.Getenv("MM_SERVER_PATH"))

	// start up docker via Mattermost server makefile
	onceStartDocker.Do(func() {
		cmd := exec.Command("make", "start-docker")
		cmd.Dir = serverPath
		stdout := &strings.Builder{}
		stderr := &strings.Builder{}
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err := cmd.Run()
		t.Log(stdout.String())
		t.Log(stderr.String())
		require.NoError(t, err, "make start-docker fail")
	})

	th := &TestHelper{}
	th.mainHelper = testlib.NewMainHelperWithOptions(&options)
	th.restoreEnv = restoreEnv

	dbStore := th.mainHelper.GetStore()
	dbStore.DropAllTables()
	dbStore.MarkSystemRanUnitTests()
	th.mainHelper.PreloadMigrations()

	store, err := New(storeWrapper{th.mainHelper}, &testLogger{t})
	require.NoError(t, err, "could not create store")
	th.Store = store

	fileBackendSettings := filestore.FileBackendSettings{
		DriverName:                         "amazons3",
		AmazonS3AccessKeyId:                "minioaccesskey",
		AmazonS3SecretAccessKey:            "miniosecretkey",
		AmazonS3Bucket:                     "mattermost-test",
		AmazonS3Region:                     "",
		AmazonS3Endpoint:                   "localhost:9000",
		AmazonS3PathPrefix:                 "",
		AmazonS3SSL:                        false,
		AmazonS3SSE:                        false,
		AmazonS3RequestTimeoutMilliseconds: 5000,
	}
	fileBackend, err := filestore.NewFileBackend(fileBackendSettings)
	require.NoError(t, err)
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

func (th *TestHelper) TearDown() {
	if th.mainHelper.SQLStore != nil {
		th.mainHelper.SQLStore.Close()
	}
	if th.mainHelper.Settings != nil {
		storetest.CleanupSqlSettings(th.mainHelper.Settings)
	}
	for k, v := range th.restoreEnv {
		_ = os.Setenv(k, v)
	}
}

func (th *TestHelper) CreateTeams(num int, namePrefix string) ([]*model.Team, error) {
	var teams []*model.Team
	for i := 0; i < num; i++ {
		team := &model.Team{
			Name:        fmt.Sprintf("%s-%d", namePrefix, i),
			DisplayName: fmt.Sprintf("%s-%d", namePrefix, i),
			Type:        model.TeamOpen,
		}
		team, err := th.mainHelper.Store.Team().Save(team)
		if err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}
	return teams, nil
}

func (th *TestHelper) CreateChannel(name string, userID string, teamID string) (*model.Channel, error) {
	channel := &model.Channel{
		Name:        name,
		DisplayName: name,
		Type:        model.ChannelTypeOpen,
		CreatorId:   userID,
		TeamId:      teamID,
	}
	return th.mainHelper.Store.Channel().Save(channel, 1024)
}

func (th *TestHelper) CreateChannels(num int, namePrefix string, userID string, teamID string) ([]*model.Channel, error) {
	var channels []*model.Channel
	for i := 0; i < num; i++ {
		channel := &model.Channel{
			Name:        fmt.Sprintf("%s-%d", namePrefix, i),
			DisplayName: fmt.Sprintf("%s-%d", namePrefix, i),
			Type:        model.ChannelTypeOpen,
			CreatorId:   userID,
			TeamId:      teamID,
		}
		channel, err := th.mainHelper.Store.Channel().Save(channel, 1024)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}
	return channels, nil
}

func (th *TestHelper) CreateChannelsWithChannelMemberHistory(num int, namePrefix string, userID string, teamID string) ([]*model.Channel, error) {
	var channels []*model.Channel
	for i := 0; i < num; i++ {
		channel := &model.Channel{
			Name:        fmt.Sprintf("%s-%d", namePrefix, i),
			DisplayName: fmt.Sprintf("%s-%d", namePrefix, i),
			Type:        model.ChannelTypeOpen,
			CreatorId:   userID,
			TeamId:      teamID,
		}
		channel, err := th.mainHelper.Store.Channel().Save(channel, 1024)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)

		err = th.mainHelper.Store.ChannelMemberHistory().LogJoinEvent(userID, channel.Id, model.GetMillis())
		if err != nil {
			return nil, err
		}
	}
	return channels, nil
}

func (th *TestHelper) CreateDirectMessageChannel(user1 *model.User, user2 *model.User) (*model.Channel, error) {
	return th.mainHelper.Store.Channel().CreateDirectChannel(user1, user2)
}

func (th *TestHelper) CreateUsers(num int, namePrefix string) ([]*model.User, error) {
	var users []*model.User
	for i := 0; i < num; i++ {
		user := &model.User{
			Username: fmt.Sprintf("%s-%d", namePrefix, i),
			Password: namePrefix,
			Email:    fmt.Sprintf("%s@example.com", model.NewId()),
		}
		user, err := th.mainHelper.Store.User().Save(user)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (th *TestHelper) CreatePosts(num int, userID string, channelID string) ([]*model.Post, error) {
	var posts []*model.Post
	for i := 0; i < num; i++ {
		post := &model.Post{
			UserId:    userID,
			ChannelId: channelID,
			Type:      model.PostTypeDefault,
			Message:   fmt.Sprintf("test post %d of %d", i, num),
		}
		post, err := th.mainHelper.Store.Post().Save(post)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, nil
}

func (th *TestHelper) CreatePostsWithAttachments(num int, userID string, channelID string) ([]*model.Post, error) {
	var posts []*model.Post
	for i := 0; i < num; i++ {
		text := "This is a test uploaded file."
		reader := strings.NewReader(text)
		size, err := th.FileBackend.WriteFile(reader, "tests/file_upload_test.txt")
		if err != nil {
			return nil, err
		}

		fileInfo := &model.FileInfo{
			Id:        model.NewId(),
			CreateAt:  model.GetMillis(),
			UpdateAt:  model.GetMillis(),
			CreatorId: userID,
			Name:      "file_upload_test.txt",
			Path:      "tests/file_upload_test.txt",
			MimeType:  "text/plain",
			Size:      size,
		}

		fileInfo, err = th.mainHelper.Store.FileInfo().Save(fileInfo)
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
		post, err = th.mainHelper.Store.Post().Save(post)
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
		reaction, err := th.mainHelper.Store.Reaction().Save(reaction)
		if err != nil {
			return nil, err
		}
		reactions = append(reactions, reaction)
	}
	return reactions, nil
}

// storeWrapper is a wrapper for MainHelper that implements SQLStoreSource interface.
type storeWrapper struct {
	mainHelper *testlib.MainHelper
}

func (sw storeWrapper) GetMasterDB() (*sql.DB, error) {
	return sw.mainHelper.SQLStore.GetInternalMasterDB(), nil
}

func (sw storeWrapper) GetReplicaDB() (*sql.DB, error) {
	// For this test helper, just return the master DB even when a replica has been asked for.
	return sw.mainHelper.SQLStore.GetInternalMasterDB(), nil
}

func (sw storeWrapper) DriverName() string {
	return *sw.mainHelper.Settings.DriverName
}

type testLogger struct {
	tb testing.TB
}

// Error logs an error message, optionally structured with alternating key, value parameters.
func (l *testLogger) Error(message string, keyValuePairs ...interface{}) {
	l.log("error", message, keyValuePairs...)
}

// Warn logs an error message, optionally structured with alternating key, value parameters.
func (l *testLogger) Warn(message string, keyValuePairs ...interface{}) {
	l.log("warn", message, keyValuePairs...)
}

// Info logs an error message, optionally structured with alternating key, value parameters.
func (l *testLogger) Info(message string, keyValuePairs ...interface{}) {
	l.log("info", message, keyValuePairs...)
}

// Debug logs an error message, optionally structured with alternating key, value parameters.
func (l *testLogger) Debug(message string, keyValuePairs ...interface{}) {
	l.log("debug", message, keyValuePairs...)
}

func (l *testLogger) log(level string, message string, keyValuePairs ...interface{}) {
	var args strings.Builder

	if len(keyValuePairs) > 0 && len(keyValuePairs)%2 != 0 {
		keyValuePairs = keyValuePairs[:len(keyValuePairs)-1]
	}

	for i := 0; i < len(keyValuePairs); i += 2 {
		args.WriteString(fmt.Sprintf("%v:%v  ", keyValuePairs[i], keyValuePairs[i+1]))
	}

	l.tb.Logf("level=%s  message=%s  %s", level, message, args.String())
}
