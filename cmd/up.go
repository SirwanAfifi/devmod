package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirwanafifi/devmod/config"
	"github.com/spf13/cobra"
)

var profile string

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start the dev workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}

		if profile == "" {
			profile = "full"
		}
		p, ok := cfg.Profiles[profile]
		if !ok {
			keys := make([]string, 0, len(cfg.Profiles))
			for k := range cfg.Profiles {
				keys = append(keys, k)
			}
			return fmt.Errorf("profile %q not found. Available: %s", profile, strings.Join(keys, ", "))
		}

		// Validate and absolutise pane dirs
		absDirs := map[string]string{}
		for key, pane := range p.Panes {
			d, err := mustAbsDir(pane.Dir)
			if err != nil {
				return fmt.Errorf("pane %q dir invalid: %w", key, err)
			}
			absDirs[key] = d
		}

		switch runtime.GOOS {
		case "darwin", "linux":
			return runTmuxGeneric(cfg.SessionName, p, absDirs)
		case "windows":
			fmt.Println("Windows detected — TODO: WSL/WezTerm fallback")
			return nil
		default:
			return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
		}
	},
}

func init() {
	upCmd.Flags().StringVarP(&profile, "profile", "p", "", "Profile to run (default: full)")
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

func shEscape(s string) string {
	// Minimal escaping for double-quoted context
	return strings.ReplaceAll(s, `"`, `\"`)
}

func runTmuxGeneric(session string, prof config.Profile, absDirs map[string]string) error {
	var b strings.Builder
	writeln := func(s string) { b.WriteString(s); b.WriteByte('\n') }

	// --- Validate we have exactly the target arrangement (2 columns; left has 2 rows) ---
	if len(prof.Layout.Columns) != 2 {
		return fmt.Errorf("expected exactly 2 columns in layout (left=[BE,FE], right=[git])")
	}
	left := prof.Layout.Columns[0]
	right := prof.Layout.Columns[1]
	if len(left.Rows) != 2 {
		return fmt.Errorf("expected left column to have exactly 2 rows (e.g. [be, fe])")
	}
	if len(right.Rows) != 1 {
		return fmt.Errorf("expected right column to have exactly 1 row (e.g. [git])")
	}
	// Resolve pane keys
	leftTopKey := left.Rows[0]
	leftBotKey := left.Rows[1]
	rightKey := right.Rows[0]

	leftTopPane, ok := prof.Panes[leftTopKey]
	if !ok {
		return fmt.Errorf("pane %q not defined", leftTopKey)
	}
	leftBotPane, ok := prof.Panes[leftBotKey]
	if !ok {
		return fmt.Errorf("pane %q not defined", leftBotKey)
	}
	rightPane, ok := prof.Panes[rightKey]
	if !ok {
		return fmt.Errorf("pane %q not defined", rightKey)
	}

	// --- Script ---
	writeln("set -e")
	writeln("tmux start-server")
	writeln(fmt.Sprintf("tmux has-session -t %q 2>/dev/null && tmux kill-session -t %q", session, session))

	// 1) Start with BE (left/top) — create window
	writeln(fmt.Sprintf(`tmux new-session -d -s %q -n %q -c %q`, session, shEscape(leftTopPane.Name), absDirs[leftTopKey]))
	writeln(fmt.Sprintf(`LEFT_TOP=$(tmux display-message -p -t %q:%s.0 '#{pane_id}')`, session, shEscape(leftTopPane.Name)))
	writeln(`tmux select-pane -T "` + shEscape(leftTopPane.Name) + `" -t "$LEFT_TOP"`)
	if strings.TrimSpace(leftTopPane.Cmd) != "" {
		writeln(fmt.Sprintf(`tmux send-keys -t "$LEFT_TOP" 'exec $SHELL -lc "%s"' C-m`, shEscape(leftTopPane.Cmd)))
	}

	// 2) Create RIGHT column (full height) from LEFT_TOP
	writeln(fmt.Sprintf(`RIGHT=$(tmux split-window -h -t "$LEFT_TOP" -c %q -P -F '#{pane_id}')`, absDirs[rightKey]))
	writeln(`tmux select-pane -T "` + shEscape(rightPane.Name) + `" -t "$RIGHT"`)
	if strings.TrimSpace(rightPane.Cmd) != "" {
		writeln(fmt.Sprintf(`tmux send-keys -t "$RIGHT" 'exec $SHELL -lc "%s"' C-m`, shEscape(rightPane.Cmd)))
	}

	// 3) Split LEFT column vertically to create FE **below** BE
	writeln(fmt.Sprintf(`LEFT_BOT=$(tmux split-window -v -t "$LEFT_TOP" -c %q -P -F '#{pane_id}')`, absDirs[leftBotKey]))
	writeln(`tmux select-pane -T "` + shEscape(leftBotPane.Name) + `" -t "$LEFT_BOT"`)
	if strings.TrimSpace(leftBotPane.Cmd) != "" {
		writeln(fmt.Sprintf(`tmux send-keys -t "$LEFT_BOT" 'exec $SHELL -lc "%s"' C-m`, shEscape(leftBotPane.Cmd)))
	}

	// 4) Make BE/FE heights equal in the LEFT column
	//    Compute half of the window height and set the bottom-left pane to that height.
	//    (tmux counts borders internally; letting tmux handle exact math is fine in practice.)
	writeln(fmt.Sprintf(`WH=$(tmux display-message -p -t %q '#{window_height}')`, session))
	writeln(`HALF=$(( WH / 2 ))`)
	writeln(`tmux resize-pane -t "$LEFT_BOT" -y "$HALF"`)

	// NOTE: do NOT call 'select-layout tiled' — it would destroy the two-column design.
	writeln(fmt.Sprintf(`tmux attach -t %q`, session))

	tmpFile := filepath.Join(os.TempDir(), "devmod_tmux.sh")
	if err := os.WriteFile(tmpFile, []byte(b.String()), 0755); err != nil {
		return err
	}

	cmd := exec.Command("/bin/sh", "-c", tmpFile)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
