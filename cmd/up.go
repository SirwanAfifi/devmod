package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/sirwanafifi/devmod/config"
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start the dev workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}

		// Resolve paths to absolute and validate they exist
		beDir, err := mustAbsDir(cfg.BackendDir)
		if err != nil {
			return fmt.Errorf("backendDir: %w", err)
		}
		feDir, err := mustAbsDir(cfg.FrontendDir)
		if err != nil {
			return fmt.Errorf("frontendDir: %w", err)
		}
		vcsDir, err := mustAbsDir(cfg.VcsDir)
		if err != nil {
			return fmt.Errorf("vcsDir: %w", err)
		}

		switch runtime.GOOS {
		case "darwin", "linux":
			return runTmux(cfg.SessionName, beDir, cfg.BackendCmd, feDir, cfg.FrontendCmd, vcsDir)
		case "windows":
			fmt.Println("Windows detected — TODO: add WSL/WezTerm fallback")
			return nil
		default:
			return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
		}
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
}

func mustAbsDir(p string) (string, error) {
	if p == "" {
		return "", errors.New("path is empty")
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", abs)
	}
	return abs, nil
}

func runTmux(session, beDir, beCmd, feDir, feCmd, vcsDir string) error {
	script := fmt.Sprintf(`
set -e

tmux start-server
tmux has-session -t "%[1]s" 2>/dev/null && tmux kill-session -t "%[1]s"

# 1) Create window with BE as the first (left, top) pane
tmux new-session -d -s "%[1]s" -n "BE" -c "%[2]s"
BE_PANE=$(tmux display-message -p -t "%[1]s:BE".0 '#{pane_id}')
tmux select-pane -T "BE" -t "$BE_PANE"
tmux send-keys -t "$BE_PANE" 'exec $SHELL -lc "%[3]s"' C-m

# 2) Create RIGHT column for git (full height)
GIT_PANE=$(tmux split-window -h -t "$BE_PANE" -c "%[6]s" -P -F '#{pane_id}')
tmux select-pane -T "git" -t "$GIT_PANE"
tmux send-keys  -t "$GIT_PANE" 'exec $SHELL -lc "lazygit"' C-m

# Optional: make right column narrower/wider (uncomment and tweak)
# tmux resize-pane -t "$GIT_PANE" -x 90   # set right column width to 90 columns

# 3) Split LEFT column vertically to create FE BELOW BE
FE_PANE=$(tmux split-window -v -t "$BE_PANE" -c "%[4]s" -P -F '#{pane_id}')
tmux select-pane -T "FE" -t "$FE_PANE"
tmux send-keys  -t "$FE_PANE" 'exec $SHELL -lc "%[5]s"' C-m

# Do NOT call 'select-layout tiled' here; it would break the 2-col design
tmux attach -t "%[1]s"
`, session, beDir, beCmd, feDir, feCmd, vcsDir)

	tmpFile := filepath.Join(os.TempDir(), "devmod_tmux.sh")
	if err := os.WriteFile(tmpFile, []byte(script), 0755); err != nil {
		return err
	}

	cmd := exec.Command("/bin/sh", "-c", tmpFile)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
