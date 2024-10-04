package cmd

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-plugin-legal-hold/processor/model"
	"github.com/mattermost/mattermost-plugin-legal-hold/processor/parse"
	"github.com/mattermost/mattermost-plugin-legal-hold/processor/view"
)

var rootCmd = &cobra.Command{
	Use:   "legalhold",
	Short: "Process a Mattermost Legal Hold",
	Long:  `Processes the data exported by the Mattermost Legal Hold plugin into a human-navigable format`,
	Run:   Process,
}

var legalHoldData string
var outputPath string
var legalHoldSecret string

func init() {
	rootCmd.PersistentFlags().StringVar(&legalHoldData, "legal-hold-data", "", "Path to the legal hold data file")
	rootCmd.PersistentFlags().StringVar(&outputPath, "output-path", "", "Path where the output files will be written")
	rootCmd.PersistentFlags().StringVar(&legalHoldSecret, "legal-hold-secret", "", "Secret to verify the legal hold data")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func Process(cmd *cobra.Command, _ []string) {
	if legalHoldData == "" {
		fmt.Println("Error: --legal-hold-data flag is required")
		fmt.Println("")
		_ = cmd.Help()
		os.Exit(1)
	}

	fmt.Println("Running the Mattermost Legal Hold Processor")
	fmt.Printf("- Input data: %s\n", legalHoldData)
	fmt.Printf("- Procesed output will be written to: %s\n", outputPath)
	fmt.Println()
	fmt.Println("Let's begin...")
	fmt.Println()

	// Extract the zip file
	fmt.Println("Extracting data to temporary directory...")

	tempPath := filepath.Join(outputPath, "temp")

	err := os.MkdirAll(tempPath, 0755)
	if err != nil {
		fmt.Printf("Error while creating temporary directory: %v\n", err)
	}

	// Clean up the temporary directory when we're done.
	defer func() {
		os.RemoveAll(tempPath)
	}()

	if err := ExtractZip(legalHoldData, tempPath); err != nil {
		fmt.Printf("Error while extracting: %v\n", err)
		os.Exit(1)
	}

	// Create a list of legal holds.
	fmt.Println("Identifying Legal Holds in output data...")
	legalHolds, err := parse.ListLegalHolds(tempPath)
	if err != nil {
		fmt.Printf("Error while listing legal holds: %v\n", err)
		os.Exit(1)
	}
	for _, hold := range legalHolds {
		fmt.Printf("- Legal Hold: %s (%s)\n", hold.Name, hold.ID)
	}
	fmt.Println()

	// Verify the legal hold data
	if legalHoldSecret != "" {
		fmt.Println("Secret key was provided, verifying legal holds...")
		var errorsOnVerify bool

		for _, hold := range legalHolds {
			fmt.Printf("- Verifying Legal Hold (%s): ", hold.Name)
			err := parse.ParseHashes(tempPath, hold.Path, legalHoldSecret)
			if err != nil {
				fmt.Printf("[Error] %v\n", err)
				errorsOnVerify = true
				continue
			} else {
				fmt.Println("Verified")
			}
		}
		fmt.Println()

		if errorsOnVerify {
			fmt.Println("Failed to verify the authenticity of the legal holds. Exiting.")
			os.Exit(1)
		}
	}

	// Process Each Legal Hold.
	for _, hold := range legalHolds {
		err = ProcessLegalHold(hold, outputPath)
		if err != nil {
			fmt.Printf("Error while processing legal hold: %v\n", err)
			os.Exit(1)
		}
	}
}

// ExtractZip extracts all files from the specified zip archive and saves them to the given output path.
func ExtractZip(zipPath string, outputPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer func() {
		if err = r.Close(); err != nil {
			fmt.Println(err.Error())
		}
	}()

	for _, f := range r.File {
		err = extractItem(f, outputPath)
		if err != nil {
			return err
		}
	}
	return nil
}

// ProcessLegalHold carries out the processing of a single legal hold within the extracted output data.
func ProcessLegalHold(hold model.LegalHold, outputPath string) error {
	fmt.Printf("Processing Legal Hold: %s\n", hold.Name)
	fmt.Println()

	index, err := parse.LoadIndex(hold)
	if err != nil {
		return err
	}

	teamLookup, channelLookup, teamForChannelLookup := parse.CreateTeamAndChannelLookup(index)

	channels, err := parse.ListChannels(hold)
	if err != nil {
		return err
	}

	fmt.Println("Finding channels...")
	for _, channel := range channels {
		fmt.Printf("- Channel: %s\n", channel.ID)
	}
	fmt.Println()

	// Build a FileID to file path lookup table.
	originalFileLookup, err := parse.ProcessFiles(hold)
	if err != nil {
		return err
	}

	// Move all attachments into position in the output folders.
	fileLookup, err := view.MoveFiles(originalFileLookup, outputPath)
	if err != nil {
		return err
	}

	for _, channel := range channels {
		posts, err := parse.LoadPosts(channel)
		if err != nil {
			return err
		}

		if posts == nil {
			continue
		}

		// Augment posts with the path to the file attachments using the fileID LUT.
		postsWithFiles := parse.AddFilesToPosts(posts, fileLookup)

		if err = view.WriteChannel(hold, channel, postsWithFiles, teamForChannelLookup[channel.ID], channelLookup[channel.ID], outputPath); err != nil {
			return err
		}
	}

	// Load data per user.
	var users []model.User
	for userID, userIndex := range index.Users {
		user := model.NewUserFromIDAndIndex(userID, userIndex)
		users = append(users, user)
		channels = parse.ListChannelsFromChannelMemberships(userIndex.Channels, hold)

		for _, channel := range channels {
			posts, err := parse.LoadPosts(channel)
			if err != nil {
				return err
			}

			postsWithFiles := parse.AddFilesToPosts(posts, fileLookup)

			if err = view.WriteUserChannel(hold, user, channel, postsWithFiles, teamForChannelLookup[channel.ID], channelLookup[channel.ID], outputPath); err != nil {
				return err
			}
		}

		allPosts := make(map[string][]*model.PostWithFiles)
		for _, channel := range channels {
			posts, err := parse.LoadPosts(channel)
			if err != nil {
				return err
			}

			postsWithFiles := parse.AddFilesToPosts(posts, fileLookup)

			allPosts[channel.ID] = postsWithFiles
		}
		if err = view.WriteUserAllChannels(hold, user, allPosts, teamForChannelLookup, channelLookup, outputPath); err != nil {
			return err
		}
	}

	if err = view.WriteIndexFile(hold, index, teamLookup, channelLookup, teamForChannelLookup, outputPath); err != nil {
		return err
	}

	return nil
}

// extractItem extracts a file from a zip archive and saves it to the specified output path.
func extractItem(f *zip.File, outputPath string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() {
		if err = rc.Close(); err != nil {
			fmt.Println(err.Error())
		}
	}()

	fpath := filepath.Join(outputPath, f.Name)
	if f.FileInfo().IsDir() {
		err := os.MkdirAll(fpath, 0644)
		if err != nil {
			return err
		}
	} else {
		fdir := filepath.Dir(fpath)
		err = os.MkdirAll(fdir, 0755)
		if err != nil {
			return err
		}

		file, err := os.Create(fpath)
		if err != nil {
			return err
		}
		defer func() {
			if err = file.Close(); err != nil {
				fmt.Println(err.Error())
			}
		}()

		_, err = io.Copy(file, rc)
		if err != nil {
			return err
		}
	}
	return nil
}
