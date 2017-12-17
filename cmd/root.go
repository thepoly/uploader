package cmd

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "uploader [command]",
	Short: "Uploader parses IDML files and turns stories into WordPress posts",
}

func init() {
	RootCmd.AddCommand(UploadCmd)
	RootCmd.AddCommand(ServerCmd)
}
