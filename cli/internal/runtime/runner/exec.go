package runner

import (
	"aide/cli/internal/platform/config"
	"aide/cli/internal/runtime/plugin"
	"aide/cli/internal/runtime/trust"
	"context"
	"fmt"
	"time"
)

func (r *Runner) resolveTLS(src config.Source) (bool, string) {
	verify := true
	caBundle := ""
	if g := r.cfg.Settings.TLS; g.VerifySSL != nil || g.CABundle != "" {
		if g.VerifySSL != nil {
			verify = *g.VerifySSL
		}
		if g.CABundle != "" {
			caBundle = g.CABundle
		}
	}
	if src.TLS != nil {
		if src.TLS.VerifySSL != nil {
			verify = *src.TLS.VerifySSL
		}
		if src.TLS.CABundle != "" {
			caBundle = src.TLS.CABundle
		}
	}
	if r.tlsVerifyOverride != nil {
		verify = *r.tlsVerifyOverride
	}
	if r.caBundleOverride != nil {
		caBundle = *r.caBundleOverride
	}
	return verify, caBundle
}

func (r *Runner) executeSource(ctx context.Context, name string, src config.Source) SourceResult {
	timeout := time.Duration(r.cfg.Settings.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	mgr := plugin.NewManager()
	pluginName := src.Plugin
	if pluginName == "" {
		pluginName = name
	}
	m, err := mgr.Get(pluginName)
	if err != nil {
		return SourceResult{
			Source:     name,
			Error:      fmt.Errorf("plugin %q not installed: %w", pluginName, err),
			DurationMs: time.Since(start).Milliseconds(),
		}
	}

	secrets, _ := plugin.ScopedSecrets(name, m)

	verifySSL, caBundle := r.resolveTLS(src)
	if verifySSL && caBundle == "" {
		caBundle = trust.SystemBundle()
	}

	req := &plugin.Request{
		Action:  "scrape",
		Config:  src.Config,
		Secrets: secrets,
		Context: map[string]any{
			"data_dir":   r.cfg.Settings.DataDir,
			"log_level":  r.logLevel,
			"log_format": r.logFormat,
			"verify_ssl": verifySSL,
			"ca_bundle":  caBundle,
		},
	}

	r.debugf("scraping %s via plugin %s", name, pluginName)
	r.infof("starting %s", name)

	resp, stderr, err := plugin.Execute(ctx, m, req)
	if stderr != "" {
		r.streamStderr(name, stderr)
	}
	if err != nil {
		r.errorf("%s failed: %v", name, err)
		return SourceResult{
			Source:     name,
			Error:      err,
			DurationMs: time.Since(start).Milliseconds(),
			Stderr:     stderr,
		}
	}
	if !resp.OK {
		msg := resp.Error
		if msg == "" {
			msg = "plugin returned ok=false"
		}
		r.errorf("%s failed: %s", name, msg)
		return SourceResult{
			Source:     name,
			Error:      fmt.Errorf("%s", msg),
			DurationMs: time.Since(start).Milliseconds(),
			Stderr:     stderr,
		}
	}

	entries := make([]ScraperEntry, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		entries = append(entries, ScraperEntry{
			Source:    name,
			Member:    e.Member,
			Category:  e.Category,
			Title:     e.Title,
			Detail:    e.Detail,
			EntryDate: e.EntryDate,
			Priority:  e.Priority,
			Metadata:  e.Metadata,
		})
	}

	for _, met := range resp.Metrics {
		entries = append(entries, ScraperEntry{
			Source:   name,
			Category: "metric",
			Title:    met.Name,
			Metadata: map[string]any{"mode": "metric", "metric_value": met.Value},
		})
	}

	teamMembers := make([]TeamMemberRaw, 0, len(resp.TeamMembers))
	for _, t := range resp.TeamMembers {
		teamMembers = append(teamMembers, TeamMemberRaw{
			Name:                t.Name,
			Email:               t.Email,
			Role:                t.Role,
			Department:          t.Department,
			Branch:              t.Branch,
			Registration:        t.Registration,
			ManagerRegistration: t.ManagerRegistration,
		})
	}

	result := SourceResult{
		Source:      name,
		Entries:     entries,
		TeamMembers: teamMembers,
		PluginResp:  resp,
		DurationMs:  time.Since(start).Milliseconds(),
		Stderr:      stderr,
	}
	r.debugf("%s finished in %dms (entries=%d)", name, result.DurationMs, len(entries))
	r.infof("%s done in %dms (%d entries)", name, result.DurationMs, len(entries))
	return result
}
