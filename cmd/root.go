package cmd

import (
	"github.com/spf13/cobra"
	"github.com/thepoly/uploader/upload"
)

var RootCmd = &cobra.Command{
	Use:   "uploader [API password] [IDML file]",
	Short: "Uploader parses IDML files and turns stories into WordPress posts",
	Run: func(cmd *cobra.Command, args []string) {
		apiPassword := args[0]
		snippetPath := args[1]
		upload.ParseAndUpload(apiPassword, snippetPath)
	},
	Args: cobra.ExactArgs(2),
}
