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

	opts := model.LegalHoldProcessOptions{
		LegalHoldData:   legalHoldData,
		OutputPath:      outputPath,
		LegalHoldSecret: legalHoldSecret,
	}

	result, err := ProcessLegalHolds(opts)
	if err != nil {
		fmt.Printf("Error processing legal holds: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully processed %d legal holds\n", len(result.LegalHolds))
	fmt.Printf("Processed %d files\n", result.FilesCount)
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

// ProcessLegalHolds processes all legal holds according to the given options
func ProcessLegalHolds(opts model.LegalHoldProcessOptions) (*model.LegalHoldProcessResult, error) {
	result := &model.LegalHoldProcessResult{
		LegalHolds: []string{},
		FilesCount: 0,
	}

	fmt.Println("Let's begin...")
	fmt.Println()

	// Extract the zip file
	fmt.Println("Extracting data to temporary directory...")

	tempPath := filepath.Join(opts.OutputPath, "temp")

	err := os.MkdirAll(tempPath, 0755)
	if err != nil {
		return nil, fmt.Errorf("error creating temporary directory: %w", err)
	}

	// Clean up the temporary directory when we're done.
	defer func() {
		os.RemoveAll(tempPath)
	}()

	if err := ExtractZip(opts.LegalHoldData, tempPath); err != nil {
		return nil, fmt.Errorf("error extracting zip: %w", err)
	}

	// Create a list of legal holds.
	fmt.Println("Identifying Legal Holds in output data...")
	legalHolds, err := parse.ListLegalHolds(tempPath)
	if err != nil {
		return nil, fmt.Errorf("error listing legal holds: %w", err)
	}
	for _, hold := range legalHolds {
		fmt.Printf("- Legal Hold: %s (%s)\n", hold.Name, hold.ID)
	}
	fmt.Println()

	// Verify the legal hold data
	if opts.LegalHoldSecret != "" {
		fmt.Println("Secret key was provided, verifying legal holds...")
		var errorsOnVerify bool

		for _, hold := range legalHolds {
			fmt.Printf("- Verifying Legal Hold (%s): ", hold.Name)
			err := parse.ParseHashes(tempPath, hold.Path, opts.LegalHoldSecret)
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
			return nil, fmt.Errorf("failed to verify the authenticity of the legal holds")
		}
	}

	// Process Each Legal Hold.
	var totalFiles int
	for _, hold := range legalHolds {
		err = ProcessLegalHold(hold, opts.OutputPath)
		if err != nil {
			return nil, fmt.Errorf("error processing legal hold %s: %w", hold.ID, err)
		}
		result.LegalHolds = append(result.LegalHolds, hold.ID)

		// Count files for this hold
		files, err := parse.ProcessFiles(hold)
		if err != nil {
			return nil, fmt.Errorf("error counting files for hold %s: %w", hold.ID, err)
		}
		totalFiles += len(files)
	}

	result.FilesCount = totalFiles
	return result, nil
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
