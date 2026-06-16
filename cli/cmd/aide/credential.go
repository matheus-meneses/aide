package main

import (
	"aide/cli/internal/runtime/plugin"
	"aide/cli/internal/security/keychain"
	"aide/cli/internal/ui/widgets"
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var credentialCmd = &cobra.Command{
	Use:   "credential",
	Short: "Manage source credentials in macOS Keychain",
}

var credentialSetCmd = &cobra.Command{
	Use:   "set <source> [key] [value]",
	Short: "Store credential fields for a source (reads plugin manifest if no key given)",
	Args:  cobra.RangeArgs(1, 3),
	RunE:  credentialSetExecute,
}

var credentialShowCmd = &cobra.Command{
	Use:   "show <source>",
	Short: "Show stored credential fields (use --reveal to see values)",
	Args:  cobra.ExactArgs(1),
	RunE:  credentialShowExecute,
}

var credentialDeleteCmd = &cobra.Command{
	Use:   "delete <source> [key]",
	Short: "Remove credentials for a source or a specific field",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  credentialDeleteExecute,
}

var credentialListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sources with stored credentials",
	RunE:  credentialListExecute,
}

func init() {
	credentialShowCmd.Flags().Bool("reveal", false, "show credential values")

	credentialCmd.AddCommand(credentialSetCmd)
	credentialCmd.AddCommand(credentialShowCmd)
	credentialCmd.AddCommand(credentialDeleteCmd)
	credentialCmd.AddCommand(credentialListCmd)

	rootCmd.AddCommand(credentialCmd)
}

func resolveCredentialSchema(mgr *plugin.Manager, source string) ([]plugin.Credential, string) {
	if m, err := mgr.Get(source); err == nil && len(m.Credentials) > 0 {
		return m.Credentials, m.Description
	}
	if source == "agent" {
		return []plugin.Credential{
			{Key: "llm_api_key", Label: "LLM API key", Secret: true},
		}, "Autonomous agent LLM endpoint"
	}
	return nil, ""
}

func credentialSetExecute(_ *cobra.Command, args []string) error {
	source := args[0]

	if len(args) >= 2 {
		key := args[1]
		var value string
		if len(args) == 3 {
			value = args[2]
		} else {
			widgets.Printf("Value for %s (hidden): ", key)
			valueBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return fmt.Errorf("reading value: %w", err)
			}
			widgets.Println()
			value = string(valueBytes)
		}
		if err := keychain.SetField(source, key, value); err != nil {
			return err
		}
		widgets.Printf("Field '%s' stored for %s\n", key, source)
		return nil
	}

	mgr := plugin.NewManager()
	creds, desc := resolveCredentialSchema(mgr, source)
	if len(creds) > 0 {
		widgets.Printf("Credentials for '%s' (%s)\n\n", source, desc)
		for _, cred := range creds {
			label := cred.Label
			if label == "" {
				label = cred.Key
			}
			var val string
			if cred.Secret {
				widgets.Printf("  %s (hidden): ", label)
				b, readErr := term.ReadPassword(int(os.Stdin.Fd()))
				widgets.Println()
				if readErr != nil {
					return fmt.Errorf("reading %s: %w", cred.Key, readErr)
				}
				val = strings.TrimSpace(string(b))
			} else {
				widgets.Printf("  %s: ", label)
				line, readErr := bufio.NewReader(os.Stdin).ReadString('\n')
				if readErr != nil {
					return fmt.Errorf("reading %s: %w", cred.Key, readErr)
				}
				val = strings.TrimSpace(line)
			}
			if val == "" {
				widgets.Printf("  (skipped)\n")
				continue
			}
			if err := keychain.SetField(source, cred.Key, val); err != nil {
				return err
			}
			widgets.Printf("  '%s' stored\n", cred.Key)
		}
		return nil
	}

	reader := bufio.NewReader(os.Stdin)
	widgets.Printf("Adding credentials for '%s'. Enter field names and values.\n", source)
	widgets.Println("Leave field name empty to finish.")
	widgets.Println()

	for {
		widgets.Print("Field name: ")
		fieldName, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		fieldName = strings.TrimSpace(fieldName)
		if fieldName == "" {
			break
		}

		widgets.Printf("Value for %s (hidden): ", fieldName)
		valueBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("reading value: %w", err)
		}
		widgets.Println()

		if err := keychain.SetField(source, fieldName, string(valueBytes)); err != nil {
			return err
		}
		widgets.Printf("  '%s' stored\n", fieldName)
	}

	cred, err := keychain.GetAll(source)
	if err != nil {
		widgets.Println("Done.")
		return nil //nolint:nilerr // credential summary after set is best-effort; keychain may lag
	}
	widgets.Printf("\nCredentials for %s: %d field(s) stored\n", source, len(cred.Fields))
	return nil
}

func credentialShowExecute(cmd *cobra.Command, args []string) error {
	source := args[0]
	reveal, _ := cmd.Flags().GetBool("reveal")

	cred, err := keychain.GetAll(source)
	if err != nil {
		return fmt.Errorf("no credentials found for %s", source)
	}

	widgets.Printf("Credential fields for %s:\n", source)
	for key, val := range cred.Fields {
		if reveal {
			widgets.Printf("  %s = %s\n", key, val)
		} else {
			widgets.Printf("  %s = ****\n", key)
		}
	}
	return nil
}

func credentialDeleteExecute(_ *cobra.Command, args []string) error {
	source := args[0]

	if len(args) == 2 {
		key := args[1]
		if err := requireConfirm(fmt.Sprintf("Remove field '%s' from %s?", key, source)); err != nil {
			return err
		}
		if err := keychain.DeleteField(source, key); err != nil {
			return err
		}
		widgets.Printf("Field '%s' removed from %s\n", key, source)
	} else {
		if err := requireConfirm(fmt.Sprintf("Remove ALL credentials for %s?", source)); err != nil {
			return err
		}
		if err := keychain.DeleteSource(source); err != nil {
			return err
		}
		widgets.Printf("All credentials removed for %s\n", source)
	}
	return nil
}

func credentialListExecute(_ *cobra.Command, _ []string) error {
	sources, err := keychain.List()
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		widgets.Println("No credentials stored.")
		return nil
	}
	widgets.Println("Sources with stored credentials:")
	for _, s := range sources {
		widgets.Printf("  %s\n", s)
	}
	return nil
}
