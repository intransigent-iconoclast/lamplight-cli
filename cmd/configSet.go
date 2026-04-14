package cmd

import (
	"fmt"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Update lamplight config.",
	Long: `Update config values:

  lamplight config set --library-path /mnt/media/books
  lamplight config set --template "{author}/{title} ({year})"

To organize audiobooks into a separate directory:

  lamplight config set --audiobook-path /mnt/media/audiobooks

If not set, audiobooks go into --library-path like everything else.

If Deluge runs in Docker, it reports paths inside the container.
Tell lamplight how to translate them to real host paths:

  lamplight config set --deluge-path /data --host-path /opt/docker/data/delugevpn/downloads

Available template tokens: {author} {title} {year} {publisher} {isbn} {format}`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		libraryPath, _ := cmd.Flags().GetString("library-path")
		template, _ := cmd.Flags().GetString("template")
		audiobookPath, _ := cmd.Flags().GetString("audiobook-path")
		delugePath, _ := cmd.Flags().GetString("deluge-path")
		hostPath, _ := cmd.Flags().GetString("host-path")

		if libraryPath == "" && template == "" && audiobookPath == "" && delugePath == "" && hostPath == "" {
			return fmt.Errorf("nothing to set — use --library-path, --audiobook-path, --template, --deluge-path, or --host-path")
		}

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewLibraryConfigRepository(db)
		cfg, err := repo.Get(ctx)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		if libraryPath != "" {
			cfg.LibraryPath = libraryPath
		}
		if template != "" {
			cfg.Template = template
		}
		if audiobookPath != "" {
			cfg.AudiobookPath = audiobookPath
		}
		if delugePath != "" {
			cfg.DelugePath = delugePath
		}
		if hostPath != "" {
			cfg.HostPath = hostPath
		}

		if err := repo.Save(ctx, cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "library-path     %s\n", cfg.LibraryPath)
		fmt.Fprintf(out, "template         %s\n", cfg.Template)
		if cfg.AudiobookPath != "" {
			fmt.Fprintf(out, "audiobook-path   %s\n", cfg.AudiobookPath)
		}
		if cfg.DelugePath != "" {
			fmt.Fprintf(out, "deluge-path      %s\n", cfg.DelugePath)
			fmt.Fprintf(out, "host-path        %s\n", cfg.HostPath)
		}

		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configSetCmd.Flags().String("library-path", "", "Root directory for your book library.")
	configSetCmd.Flags().String("template", "", "Subfolder template. Tokens: {author} {title} {year} {publisher} {isbn} {format}")
	configSetCmd.Flags().String("audiobook-path", "", "Separate root for audiobooks. If not set, audiobooks go into --library-path.")
	configSetCmd.Flags().String("deluge-path", "", "Path prefix Deluge reports (inside container), e.g. /data")
	configSetCmd.Flags().String("host-path", "", "Actual host path that maps to --deluge-path, e.g. /opt/docker/data/delugevpn/downloads")
}
