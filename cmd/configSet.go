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
	Long: `Update one or both config values:

  lamplight config set --library-path ~/lamplight
  lamplight config set --template "{author}/{title} ({year})"

Available template tokens: {author} {title} {year} {publisher} {isbn} {format}`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		libraryPath, _ := cmd.Flags().GetString("library-path")
		template, _ := cmd.Flags().GetString("template")

		if libraryPath == "" && template == "" {
			return fmt.Errorf("nothing to set — use --library-path and/or --template")
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

		if err := repo.Save(ctx, cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "library-path  %s\n", cfg.LibraryPath)
		fmt.Fprintf(out, "template      %s\n", cfg.Template)

		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configSetCmd.Flags().String("library-path", "", "Root directory for your book library.")
	configSetCmd.Flags().String("template", "", "Subfolder template. Tokens: {author} {title} {year} {publisher} {isbn} {format}")
}
