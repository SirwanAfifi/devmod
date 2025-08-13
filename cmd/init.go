package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise a new project config",
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
		path := ".devmod.yml"
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists", path)
		}
		if err := os.WriteFile(path, []byte(template), 0644); err != nil {
			return err
		}
		abs, _ := filepath.Abs(path)
		fmt.Printf("Created %s\n", abs)
		return nil
	},
}

func init() { rootCmd.AddCommand(initCmd) }
