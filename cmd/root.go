package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "lamplight",
	Short: "find and download books from your indexers.",
	Long: `lamplight is a CLI for finding and downloading books.

it talks to your self-hosted Prowlarr or Jackett instance, searches across
all your configured indexers, pipes results straight to Deluge, tracks your
downloads, and organizes completed files into your library.

quick start:

  lamplight provider add --name prowlarr --type prowlarr --host localhost --port 9696 --api-key xxx
  lamplight provider sync
  lamplight client add --name deluge --type deluge --host localhost --port 8112 --password xxx
  lamplight config set --library-path /mnt/media/books
  lamplight search "dune" -t epub
  lamplight download 1
  lamplight history sync
  lamplight organize`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {}
