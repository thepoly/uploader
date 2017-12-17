package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thepoly/uploader/server"
)

var ServerCmd = &cobra.Command{
	Use:   "server [API password] [IDML file]",
	Short: "run the server",
	Run: func(cmd *cobra.Command, args []string) {
		apiPassword := args[0]
		server, err := server.New(apiPassword)
		if err != nil {
			fmt.Fprint(os.Stderr, "Unable to create server:", err.Error())
			return
		}
		server.Run()
	},
	Args: cobra.ExactArgs(1),
}
