package cmd

import (
	"fmt"
	"os"

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

func Process(cmd *cobra.Command, args []string) {
	fmt.Println("Running the Mattermost Legal Hold Processor")
	fmt.Printf("- Input data: %s\n", legalHoldData)
	fmt.Printf("- Procesed output will be written to: %s\n", outputPath)
	fmt.Println()
	fmt.Println("Let's begin...")
	fmt.Println()
}
