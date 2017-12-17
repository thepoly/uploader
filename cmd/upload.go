package cmd

import (
	"github.com/spf13/cobra"
	"github.com/thepoly/uploader/upload"
)

var UploadCmd = &cobra.Command{
	Use:   "upload [API password] [IDML file]",
	Short: "upload an IDML file",
	Run: func(cmd *cobra.Command, args []string) {
		apiPassword := args[0]
		snippetPath := args[1]
		upload.ParseAndUpload(apiPassword, snippetPath)
	},
	Args: cobra.ExactArgs(2),
}
