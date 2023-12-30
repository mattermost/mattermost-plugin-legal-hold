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

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func Process(cmd *cobra.Command, args []string) {
	fmt.Println("Process a file...")
}
