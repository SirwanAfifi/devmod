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
		// Example minimal config template
		template := `sessionName: devmod-demo
backendDir: ./backend
backendCmd: dotnet watch run
frontendDir: ./frontend
frontendCmd: pnpm dev
vcsDir: .
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

func init() {
	rootCmd.AddCommand(initCmd)
}
