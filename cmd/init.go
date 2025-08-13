package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	initProject bool // if true, write ./.devmod.yml instead of global config
	initForce   bool // if true, overwrite existing file
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise a devmod config",
	Long: `Initialise a devmod config.

By default this writes your global config to:
  $HOME/.config/devmod/config.yml

Use --project to write a per-project override:
  ./.devmod.yml
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		template := `version: 1
sessionName: your-project-name
profiles:
  full:
    layout:
      columns:
        - rows: ["be", "fe"]
        - rows: ["git"]
    panes:
      be:
        name: "BE"
        dir: "/path/to/backend"
        cmd: "dotnet watch run"
      fe:
        name: "FE"
        dir: "/path/to/frontend"
        cmd: "npm start"
      git:
        name: "git"
        dir: "/path/to/project"
        cmd: "lazygit"
`

		var target string
		if initProject {
			target = ".devmod.yml"
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cannot resolve home directory: %w", err)
			}
			dir := filepath.Join(home, ".config", "devmod")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("creating %s: %w", dir, err)
			}
			target = filepath.Join(dir, "config.yml")
		}

		if _, err := os.Stat(target); err == nil && !initForce {
			return fmt.Errorf("%s already exists (use --force to overwrite)", target)
		}

		if err := os.WriteFile(target, []byte(template), 0o644); err != nil {
			return err
		}

		abs, _ := filepath.Abs(target)
		fmt.Printf("Created %s\n", abs)
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&initProject, "project", false, "write a project override (.devmod.yml) in the current directory")
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing file")
	rootCmd.AddCommand(initCmd)
}
