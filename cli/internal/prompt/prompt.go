package prompt

import (
	"aide/cli/internal/config"
	"aide/cli/internal/keychain"
	"aide/cli/internal/plugin"
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

func PickPlugin(mgr *plugin.Manager, configured map[string]config.Source) (string, error) {
	manifests, err := mgr.List()
	if err != nil {
		return "", fmt.Errorf("listing plugins: %w", err)
	}

	var available []string
	for _, m := range manifests {
		if _, exists := configured[m.Name]; !exists {
			label := m.Name
			if m.Description != "" {
				label += " - " + m.Description
			}
			available = append(available, label)
		}
	}

	if len(available) == 0 {
		fmt.Println("All installed plugins are already configured.")
		fmt.Println("\nYour configured sources:")
		for name := range configured {
			fmt.Printf("  - %s\n", name)
		}
		fmt.Println("\nInstall new plugins with:")
		fmt.Println("  aide plugin install <name>")
		return "", fmt.Errorf("nothing to add")
	}

	var choice string
	err = survey.AskOne(&survey.Select{
		Message: "Select a plugin to configure:",
		Options: available,
	}, &choice)
	if err != nil {
		return "", err
	}

	return strings.SplitN(choice, " - ", 2)[0], nil
}

func ConfigurePlugin(m *plugin.Manifest) (map[string]any, error) {
	cfg := make(map[string]any)

	for _, field := range m.Config {
		if !field.Required && field.Default == "" {
			var confirm bool
			_ = survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf("Configure %s?", field.Label),
				Default: false,
			}, &confirm)
			if !confirm {
				continue
			}
		}

		var opts []survey.AskOpt
		if field.Required && field.Default == "" {
			opts = append(opts, survey.WithValidator(survey.Required))
		}

		var value string
		if err := survey.AskOne(&survey.Input{
			Message: field.Label,
			Default: field.Default,
		}, &value, opts...); err != nil {
			return nil, err
		}

		if value == "" && field.Default != "" {
			value = field.Default
		}
		if value != "" {
			cfg[field.Key] = value
		}
	}

	return cfg, nil
}

func SetupPluginCredentials(m *plugin.Manifest, sourceName string) error {
	if len(m.Credentials) == 0 {
		return nil
	}

	fmt.Println("\nCredentials:")

	for _, cred := range m.Credentials {
		var value string
		var err error

		if cred.Secret {
			err = survey.AskOne(&survey.Password{
				Message: cred.Label,
			}, &value, survey.WithValidator(survey.Required))
		} else {
			err = survey.AskOne(&survey.Input{
				Message: cred.Label,
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
