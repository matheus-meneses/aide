package prompt

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"

	"aide/cli/internal/config"
	"aide/cli/internal/keychain"
	"aide/cli/internal/registry"
)

func PickSource(reg *registry.Registry, configured map[string]config.Source) (string, error) {
	names := reg.ListSources()
	var available []string
	for _, name := range names {
		if _, exists := configured[name]; !exists {
			src := reg.Sources[name]
			available = append(available, name+" - "+src.Description)
		}
	}

	if len(available) == 0 {
		fmt.Println("All sources from the registry are already configured.")
		fmt.Println("\nYour configured sources:")
		for name := range configured {
			fmt.Printf("  - %s\n", name)
		}
		fmt.Println("\nYou can manage them with:")
		fmt.Println("  aide config source list       Show status")
		fmt.Println("  aide config source disable    Disable a source")
		fmt.Println("  aide config source remove     Remove and re-add")
		fmt.Println("  aide config check             Verify all sources")
		return "", fmt.Errorf("nothing to add")
	}

	var choice string
	err := survey.AskOne(&survey.Select{
		Message: "Select a source to configure:",
		Options: available,
	}, &choice)
	if err != nil {
		return "", err
	}

	return strings.SplitN(choice, " - ", 2)[0], nil
}

func ConfigureSource(def *registry.SourceDef) (map[string]any, error) {
	cfg := make(map[string]any)

	for _, field := range def.Fields {
		if !field.Required && field.Default == "" {
			var confirm bool
			survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf("Configure %s?", field.Label),
				Default: false,
			}, &confirm)
			if !confirm {
				continue
			}
		}

		value, err := promptField(field)
		if err != nil {
			return nil, err
		}
		if value == "" && field.Default != "" {
			value = field.Default
		}
		if value == "" {
			continue
		}

		if field.Type == "json" {
			var parsed any
			if json.Unmarshal([]byte(value), &parsed) == nil {
				cfg[field.Key] = parsed
				continue
			}
		}

		if field.Type == "list" {
			parts := strings.Split(value, ",")
			trimmed := make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					trimmed = append(trimmed, p)
				}
			}
			cfg[field.Key] = trimmed
			continue
		}

		cfg[field.Key] = value
	}

	return cfg, nil
}

func promptField(field registry.Field) (string, error) {
	msg := field.Label
	if field.Hint != "" {
		msg += " (" + field.Hint + ")"
	}

	var value string
	q := &survey.Input{
		Message: msg,
		Default: field.Default,
	}

	var opts []survey.AskOpt
	if field.Required && field.Default == "" {
		opts = append(opts, survey.WithValidator(survey.Required))
	}

	if err := survey.AskOne(q, &value, opts...); err != nil {
		return "", err
	}
	return value, nil
}

func SetupCredentials(def *registry.SourceDef, sourceName string) error {
	if len(def.Credentials) == 0 {
		return nil
	}

	fmt.Println("\nCredentials:")

	for _, cred := range def.Credentials {
		var value string
		var err error

		if cred.Secret {
			msg := cred.Label
			if cred.Hint != "" {
				msg += " (" + cred.Hint + ")"
			}
			err = survey.AskOne(&survey.Password{
				Message: msg,
			}, &value, survey.WithValidator(survey.Required))
		} else {
			msg := cred.Label
			if cred.Hint != "" {
				msg += " (" + cred.Hint + ")"
			}
			err = survey.AskOne(&survey.Input{
				Message: msg,
			}, &value, survey.WithValidator(survey.Required))
		}

		if err != nil {
			return err
		}

		if err := keychain.SetField(sourceName, cred.Key, value); err != nil {
			return fmt.Errorf("storing credential '%s': %w", cred.Key, err)
		}
	}

	return nil
}
