package cmd

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "legalhold",
	Short: "Process a Mattermost Legal Hold",
	Long:  `Processes the data exported by the Mattermost Legal Hold plugin into a human-navigable format`,
	Run:   Process,
}

var legalHoldData string
var outputPath string

func init() {
	rootCmd.PersistentFlags().StringVar(&legalHoldData, "legal-hold-data", "", "Path to the legal hold data file")
	rootCmd.PersistentFlags().StringVar(&outputPath, "output-path", "", "Path where the output files will be written")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func Process(_ *cobra.Command, _ []string) {
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

	if err := ExtractZip(legalHoldData, tempPath); err != nil {
		fmt.Printf("Error while extracting: %v\n", err)
		os.Exit(1)
	}

	// Create a list of legal holds.
	fmt.Println("Identifying Legal Holds in output data...")
	legalHolds, err := listLegalHolds(tempPath)
	if err != nil {
		fmt.Printf("Error while listing legal holds: %v\n", err)
		os.Exit(1)
	}
	for _, hold := range legalHolds {
		fmt.Printf("Legal Hold: %s (%s)\n", hold.name, hold.id)
	}
	fmt.Println()
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

type LegalHold struct {
	path string
	name string
	id   string
}

// listLegalHolds retrieves a list of LegalHold objects from the specified directory path
// containing an unpacked legal hold export.
func listLegalHolds(tempPath string) ([]LegalHold, error) {
	legalHoldsPath := filepath.Join(tempPath, "legal_hold")

	files, err := os.ReadDir(legalHoldsPath)
	if err != nil {
		return nil, err
	}

	var legalHolds []LegalHold
	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		nameID := strings.Split(file.Name(), "_(")
		if len(nameID) != 2 || !strings.HasSuffix(nameID[1], ")") {
			return nil, errors.New("directory name does not match pattern name_(id)")
		}

		id := strings.TrimSuffix(nameID[1], ")")
		legalHolds = append(legalHolds, LegalHold{path: filepath.Join(legalHoldsPath, file.Name()), name: nameID[0], id: id})
	}

	return legalHolds, nil
}
