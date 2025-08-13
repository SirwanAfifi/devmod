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

func checkRequiredTools(cmds []string) error {
	missing := []string{}
	for _, c := range cmds {
		if _, err := exec.LookPath(c); err != nil {
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required tools: %s", strings.Join(missing, ", "))
	}
	return nil
}

func extractCommandsFromProfile(p config.Profile) []string {
	tools := map[string]struct{}{"tmux": {}} // tmux is always required
	for _, pane := range p.Panes {
		if pane.Cmd == "" {
			continue
		}
		// Take the first word from the command string
		fields := strings.Fields(pane.Cmd)
		if len(fields) > 0 {
			tools[fields[0]] = struct{}{}
		}
	}
	list := make([]string, 0, len(tools))
	for t := range tools {
		list = append(list, t)
	}
	return list
}

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

		// Pre-flight: check required tools
		required := extractCommandsFromProfile(p)
		if err := checkRequiredTools(required); err != nil {
			return err
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

func runTmuxGeneric(session string, prof config.Profile, absDirs map[string]string) error {
	var b strings.Builder
	writeln := func(s string) { b.WriteString(s); b.WriteByte('\n') }

	if len(prof.Layout.Columns) == 0 {
		return fmt.Errorf("layout.columns must have at least one column")
	}
	// Validate keys
	for ci, col := range prof.Layout.Columns {
		if len(col.Rows) == 0 {
			return fmt.Errorf("layout.columns[%d] must have at least one row", ci)
		}
		for _, key := range col.Rows {
			if _, ok := prof.Panes[key]; !ok {
				return fmt.Errorf("pane %q referenced in layout but not defined in panes", key)
			}
		}
	}

	writeln("set -e")
	writeln("tmux start-server")
	writeln(fmt.Sprintf("tmux has-session -t %q 2>/dev/null && tmux kill-session -t %q", session, session))

	// ----- 1) Create the very first pane: column 0, row 0
	c0r0Key := prof.Layout.Columns[0].Rows[0]
	c0r0Pane := prof.Panes[c0r0Key]
	writeln(fmt.Sprintf(`tmux new-session -d -s %q -n %q -c %q`, session, shEscape(c0r0Pane.Name), absDirs[c0r0Key]))
	writeln(fmt.Sprintf(`C0R0=$(tmux display-message -p -t %q:%s.0 '#{pane_id}')`, session, shEscape(c0r0Pane.Name)))
	writeln(`tmux select-pane -T "` + shEscape(c0r0Pane.Name) + `" -t "$C0R0"`)
	if strings.TrimSpace(c0r0Pane.Cmd) != "" {
		writeln(fmt.Sprintf(`tmux send-keys -t "$C0R0" 'exec $SHELL -lc "%s"' C-m`, shEscape(c0r0Pane.Cmd)))
	}

	// ----- 2) Create the TOP panes for columns 1..N-1 by splitting horizontally off C0R0
	// This guarantees every right column is full-height.
	for c := 1; c < len(prof.Layout.Columns); c++ {
		key0 := prof.Layout.Columns[c].Rows[0]
		p0 := prof.Panes[key0]
		dir0 := absDirs[key0]
		writeln(fmt.Sprintf(`C%[1]dR0=$(tmux split-window -h -t "$C0R0" -c %q -P -F '#{pane_id}')`, c, dir0))
		writeln(fmt.Sprintf(`tmux select-pane -T "%s" -t "$C%dR0"`, shEscape(p0.Name), c))
		if strings.TrimSpace(p0.Cmd) != "" {
			writeln(fmt.Sprintf(`tmux send-keys -t "$C%dR0" 'exec $SHELL -lc "%s"' C-m`, c, shEscape(p0.Cmd)))
		}
	}

	// ----- 3) For each column, stack remaining rows vertically under that column's top
	for c := 0; c < len(prof.Layout.Columns); c++ {
		rows := prof.Layout.Columns[c].Rows
		if len(rows) == 1 {
			continue
		}
		// base is the top pane id of this column
		base := fmt.Sprintf(`"$C%dR0"`, c)
		for r := 1; r < len(rows); r++ {
			key := rows[r]
			p := prof.Panes[key]
			dir := absDirs[key]
			writeln(fmt.Sprintf(`C%[1]dR%[2]d=$(tmux split-window -v -t %s -c %q -P -F '#{pane_id}')`, c, r, base, dir))
			writeln(fmt.Sprintf(`tmux select-pane -T "%s" -t "$C%dR%d"`, shEscape(p.Name), c, r))
			if strings.TrimSpace(p.Cmd) != "" {
				writeln(fmt.Sprintf(`tmux send-keys -t "$C%dR%d" 'exec $SHELL -lc "%s"' C-m`, c, r, shEscape(p.Cmd)))
			}
			base = fmt.Sprintf(`"$C%dR%d"`, c, r)
		}
	}

	// ----- 4) Equalise heights within each column that has >1 row (keeps widths intact)
	writeln(fmt.Sprintf(`WH=$(tmux display-message -p -t %q '#{window_height}')`, session))
	for c := 0; c < len(prof.Layout.Columns); c++ {
		rows := prof.Layout.Columns[c].Rows
		if len(rows) <= 1 {
			continue
		}
		writeln(fmt.Sprintf(`H%d=$(( WH / %d ))`, c, len(rows)))
		for r := len(rows) - 1; r >= 0; r-- {
			writeln(fmt.Sprintf(`tmux resize-pane -t "$C%dR%d" -y "$H%d"`, c, r, c))
		}
	}

	// (Optional) equalise column widths or set specific widths here if you add config.

	// ----- 5) Attach (do NOT use 'tiled')
	writeln(fmt.Sprintf(`tmux attach -t %q`, session))

	tmpFile := filepath.Join(os.TempDir(), "devmod_tmux.sh")
	if err := os.WriteFile(tmpFile, []byte(b.String()), 0755); err != nil {
		return err
	}

	cmd := exec.Command("/bin/sh", "-c", tmpFile)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func shEscape(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}
