package agent

import (
	"aide/cli/internal/platform/config"
	"embed"
	"fmt"
	"strings"
)

// promptFS holds the static, prose system-prompt layers. Keeping them as
// Markdown (rather than Go string literals) makes the prompt easy to read,
// diff, and edit without escaping. The dynamic pieces stay in Go
// (renderPreferences) and the safety guardrail stays a code constant in
// guardrail.go on purpose, as the security anchor.
//
//go:embed prompts/*.md
var promptFS embed.FS

var (
	promptPrecedencePreamble = mustPrompt("precedence.md")
	chatPrecedenceNote       = mustPrompt("chat_note.md")
	agentRolePrompt          = mustPrompt("role.md")
	agentCoreRules           = mustPrompt("core_rules.md")
	agentDefaultBehavior     = mustPrompt("default_behavior.md")
)

// mustPrompt loads an embedded prompt segment, trimming surrounding whitespace
// so the assembled prompt spacing is controlled by the caller. It panics if a
// segment is missing, surfacing the error at startup/test time.
func mustPrompt(name string) string {
	b, err := promptFS.ReadFile("prompts/" + name)
	if err != nil {
		panic("agent: missing embedded prompt " + name + ": " + err.Error())
	}
	return strings.TrimSpace(string(b))
}

// renderPreferences turns structured preferences into directive sentences for
// the USER PREFERENCES layer. When includeBehavior is false (chat path) only
// the tone directive is emitted, since notification policy applies to the
// autonomous loop. An empty result means "no override; defaults stand".
func renderPreferences(p config.AgentPreferences, includeBehavior bool) string {
	var lines []string

	if includeBehavior {
		switch p.NotificationLevel() {
		case config.NotifySilent:
			lines = append(lines, "Do not send notifications this cycle unless something is a genuine emergency; prefer the activity feed via send_message.")
		case config.NotifyNormal:
			lines = append(lines, "Notify the user about important changes, not only urgent ones; routine items can still go to the activity feed.")
		case config.NotifyAll:
			lines = append(lines, "Notify the user about all noteworthy changes, not only urgent ones. The default 'urgent only' restriction does not apply.")
		}
		if p.MaxNotificationsPerCycle > 0 {
			lines = append(lines, fmt.Sprintf("Send at most %d notification(s) this cycle.", p.MaxNotificationsPerCycle))
		}
	}

	if tone := strings.TrimSpace(p.Tone); tone != "" {
		lines = append(lines, fmt.Sprintf("Communicate in a %s tone.", tone))
	}

	return strings.Join(lines, "\n")
}
