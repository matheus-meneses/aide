package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var ansiRe = regexp.MustCompile(`\x1b\][^\x1b]*\x1b\\|\x1b\[[0-9;]*[a-zA-Z]`)

type ExecResult struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
	Text string      `json:"text,omitempty"`
}

func (a *Agent) ExecuteCommand(ctx context.Context, command string) *ExecResult {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return &ExecResult{Type: "text", Text: "empty command"}
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "memory":
		return a.execMemory()
	case "status":
		return a.execStatus()
	case "today":
		return a.execToday()
	case "items":
		return a.execItems(args)
	case "scrape":
		return a.execScrape(ctx, args)
	case "health":
		return a.execHealth()
	case "whoami":
		return a.execWhoami(args)
	case "ack":
		return a.execAck(args)
	case "prune":
		return a.execPrune(args)
	case "stats":
		return a.execStats()
	case "team":
		return a.execTeam(args)
	case "command":
		return a.execCLI(ctx, args)
	default:
		return a.execCLI(ctx, append([]string{cmd}, args...))
	}
}

func (a *Agent) execMemory() *ExecResult {
	mem, err := a.store.Memory.LoadLast()
	if err != nil {
		return &ExecResult{Type: "memory", Text: "No memory stored yet. The agent will save its first memory after the next cycle."}
	}
	return &ExecResult{Type: "memory", Data: mem}
}

func (a *Agent) execStatus() *ExecResult {
	return &ExecResult{Type: "status", Data: a.StatusSnapshot()}
}

func (a *Agent) execToday() *ExecResult {
	events, err := a.store.Items.TodayEvents()
	if err != nil {
		return &ExecResult{Type: "schedule", Text: fmt.Sprintf("error: %v", err)}
	}
	return &ExecResult{Type: "schedule", Data: events}
}

func (a *Agent) execItems(args []string) *ExecResult {
	source := ""
	for i, arg := range args {
		if arg == "--source" && i+1 < len(args) {
			source = args[i+1]
			break
		}
		if !strings.HasPrefix(arg, "-") && source == "" {
			source = arg
		}
	}

	items, err := a.store.Items.QueryOpen(source, "", "")
	if err != nil {
		return &ExecResult{Type: "items", Text: fmt.Sprintf("error: %v", err)}
	}
	return &ExecResult{Type: "items", Data: items}
}

func (a *Agent) execScrape(ctx context.Context, args []string) *ExecResult {
	var sources []string
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			sources = append(sources, arg)
		}
	}

	result, err := a.runScrape(ctx, sources)
	if err != nil {
		return &ExecResult{Type: "text", Text: fmt.Sprintf("scrape failed: %v", err)}
	}

	data := map[string]interface{}{
		"sources_total":  result.SourcesTotal,
		"sources_ok":     result.SourcesOK,
		"sources_failed": result.SourcesFailed,
	}
	return &ExecResult{Type: "scrape", Data: data}
}

func (a *Agent) execHealth() *ExecResult {
	health, err := a.store.Runs.AllHealth()
	if err != nil {
		return &ExecResult{Type: "status", Text: fmt.Sprintf("error: %v", err)}
	}
	return &ExecResult{Type: "status", Data: map[string]interface{}{"health": health}}
}

func (a *Agent) execAck(args []string) *ExecResult {
	if len(args) == 0 {
		return &ExecResult{Type: "text", Text: "Usage: /ack <fingerprint> [title]"}
	}
	fingerprint := args[0]
	title := fingerprint
	if len(args) > 1 {
		title = strings.Join(args[1:], " ")
	}
	if err := a.store.Acks.Add(fingerprint, title); err != nil {
		return &ExecResult{Type: "text", Text: fmt.Sprintf("error: %v", err)}
	}
	return &ExecResult{Type: "text", Text: fmt.Sprintf("Acknowledged: %s", title)}
}

func (a *Agent) execPrune(args []string) *ExecResult {
	days := 7
	if len(args) > 0 {
		if d, err := strconv.Atoi(args[0]); err == nil && d > 0 {
			days = d
		}
	}
	result, err := a.store.Maintenance.Prune(days)
	if err != nil {
		return &ExecResult{Type: "text", Text: fmt.Sprintf("prune error: %v", err)}
	}
	text := fmt.Sprintf("Pruned (kept %d days):\n- %d items\n- %d messages\n- %d sessions\n- %d memories\n- %d metrics\n- %d runs\n- %d acks\n- %d token records",
		days, result.Items, result.Messages, result.Sessions, result.Memories, result.Metrics, result.Runs, result.Acks, result.Tokens)
	return &ExecResult{Type: "text", Text: text}
}

func (a *Agent) execStats() *ExecResult {
	summary, err := a.store.Tokens.Stats()
	if err != nil {
		return &ExecResult{Type: "text", Text: fmt.Sprintf("error: %v", err)}
	}
	daily, _ := a.store.Tokens.DailyStats(7)
	data := map[string]interface{}{
		"summary": summary,
		"daily":   daily,
	}
	return &ExecResult{Type: "stats", Data: data}
}

func (a *Agent) execWhoami(args []string) *ExecResult {
	if len(args) > 0 && args[0] == "set" {
		parts := args[1:]
		if len(parts) < 2 {
			return &ExecResult{Type: "text", Text: "Usage: /whoami set <full name> <email> [nickname]\n\nExample: /whoami set John Doe john@company.com John"}
		}

		var name, email, preferred string
		for i, p := range parts {
			if strings.Contains(p, "@") {
				name = strings.Join(parts[:i], " ")
				email = p
				if i+1 < len(parts) {
					preferred = strings.Join(parts[i+1:], " ")
				}
				break
			}
		}
		if email == "" {
			name = parts[0]
			email = parts[1]
			if len(parts) > 2 {
				preferred = strings.Join(parts[2:], " ")
			}
		}
		if err := a.store.Profile.SetIdentity(name, email, preferred); err != nil {
			return &ExecResult{Type: "text", Text: fmt.Sprintf("failed to save identity: %v", err)}
		}
		if preferred == "" {
			if fields := strings.Fields(name); len(fields) > 0 {
				preferred = fields[0]
			} else {
				preferred = "there"
			}
		}

		return &ExecResult{Type: "text", Text: fmt.Sprintf("Identity saved! Hi %s.", preferred)}
	}

	profile, err := a.store.Profile.All()
	if err != nil || len(profile) == 0 {
		return &ExecResult{Type: "text", Text: "No identity configured. Use: `/whoami set <name> <email> [nickname]`\n\nExample: `/whoami set Matheus Silva matheus@inter.com Matheus`"}
	}
	return &ExecResult{Type: "text", Text: fmt.Sprintf("**Name:** %s\n**Email:** %s\n**Nickname:** %s", profile["name"], profile["email"], profile["preferred_name"])}
}

func (a *Agent) execTeam(args []string) *ExecResult {
	view := "tree"
	source := ""
	for i, arg := range args {
		switch arg {
		case "flat":
			view = "flat"
		case "tree":
			view = "tree"
		case "--view":
			if i+1 < len(args) {
				view = args[i+1]
			}
		case "--source":
			if i+1 < len(args) {
				source = args[i+1]
			}
		}
	}

	members, err := a.store.Team.All()
	if err != nil {
		return &ExecResult{Type: "text", Text: fmt.Sprintf("error: %v", err)}
	}

	if source != "" {
		filtered := members[:0]
		for _, m := range members {
			if m.Source == source {
				filtered = append(filtered, m)
			}
		}
		members = filtered
	}

	if len(members) == 0 {
		return &ExecResult{Type: "text", Text: "No team members found."}
	}

	return &ExecResult{
		Type: "team",
		Data: map[string]interface{}{
			"members": members,
			"view":    view,
		},
	}
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}

// aideCLIPath resolves the `aide` CLI binary without ever returning the desktop
// (aide-app) binary, which would relaunch the GUI instead of running a command.
func aideCLIPath() string {
	if exe, err := os.Executable(); err == nil && exe != "" {
		if sibling := filepath.Join(filepath.Dir(exe), "aide-cli"); isExecutableFile(sibling) {
			return sibling
		}
		if filepath.Base(exe) == "aide" {
			return exe
		}
	}
	if p, err := exec.LookPath("aide"); err == nil {
		return p
	}
	if home, err := os.UserHomeDir(); err == nil {
		if cand := filepath.Join(home, ".local", "bin", "aide"); isExecutableFile(cand) {
			return cand
		}
	}
	return ""
}

func (a *Agent) execCLI(ctx context.Context, args []string) *ExecResult {
	if len(args) == 0 {
		return &ExecResult{Type: "text", Text: "Usage: /command <subcommand> [args]\n\nExample: /command report"}
	}

	bin := aideCLIPath()
	if bin == "" {
		return &ExecResult{Type: "text", Text: "The `aide` CLI was not found. Install it (e.g. to ~/.local/bin/aide) to run `/command`."}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(timeoutCtx, bin, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if output == "" {
		output = stderr.String()
	}
	if err != nil && output == "" {
		output = fmt.Sprintf("error: %v", err)
	}

	output = ansiRe.ReplaceAllString(output, "")

	return &ExecResult{Type: "text", Text: strings.TrimSpace(output)}
}
