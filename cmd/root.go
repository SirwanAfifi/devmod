package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "devmod",
	Short: "DevMod - spin up your development tmux workspace",
	Long:  `DevMod is a CLI tool to launch a tmux session with your custom setup`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
