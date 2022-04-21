package cmd

import (
	"fmt"

	"github.com/lunemec/eve-accountant/pkg/version"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print accountantBot version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.VersionString)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
