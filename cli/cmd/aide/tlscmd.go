package main

import (
	"aide/cli/internal/platform/config"
	"aide/cli/internal/platform/xdg"
	"aide/cli/internal/ui/widgets"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	tlsFetchSource string
	tlsFetchGlobal bool
	tlsFetchOutput string
)

var tlsCmd = &cobra.Command{
	Use:   "tls",
	Short: "Inspect and trust TLS certificates for sources",
}

var tlsFetchCmd = &cobra.Command{
	Use:   "fetch <host[:port]>",
	Short: "Fetch a server's certificate chain and store it as a CA bundle",
	Long: "Connects to the host, saves the presented certificate chain as a PEM bundle, " +
		"and prints each certificate's SHA-256 fingerprint.\n\n" +
		"WARNING: this is trust-on-first-use. The chain is fetched over a connection that " +
		"is NOT yet verified, so an attacker on the path could supply their own certificate. " +
		"Always confirm the printed fingerprint with your IT/security team before trusting it. " +
		"Prefer installing the CA into your OS trust store, which aide reads automatically.",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          tlsFetchExecute,
}

func init() {
	tlsFetchCmd.Flags().StringVar(&tlsFetchSource, "source", "", "wire the bundle into this source's tls.ca_bundle")
	tlsFetchCmd.Flags().BoolVar(&tlsFetchGlobal, "global", false, "wire the bundle into settings.tls.ca_bundle")
	tlsFetchCmd.Flags().StringVar(&tlsFetchOutput, "output", "", "output PEM path (default ~/.aide/certs/<host>.pem)")
	tlsCmd.AddCommand(tlsFetchCmd)
	rootCmd.AddCommand(tlsCmd)
}

func tlsFetchExecute(_ *cobra.Command, args []string) error {
	host, addr := parseTLSTarget(args[0])

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // intentional: TOFU fetch of an untrusted chain
		ServerName:         host,
	})
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", addr, err)
	}
	defer conn.Close()

	chain := conn.ConnectionState().PeerCertificates
	if len(chain) == 0 {
		return fmt.Errorf("%s presented no certificates", addr)
	}

	outPath, err := resolveCertOutput(host)
	if err != nil {
		return err
	}

	var buf strings.Builder
	for _, cert := range chain {
		if err := pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}); err != nil {
			return fmt.Errorf("encoding certificate: %w", err)
		}
	}
	if err := os.WriteFile(outPath, []byte(buf.String()), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}

	widgets.Printf("Saved %d certificate(s) from %s to %s\n\n", len(chain), addr, outPath)
	for i, cert := range chain {
		role := "intermediate"
		switch {
		case i == 0:
			role = "leaf"
		case cert.IsCA && string(cert.RawSubject) == string(cert.RawIssuer):
			role = "root"
		}
		widgets.Printf("  [%s] %s\n", role, cert.Subject.String())
		widgets.Printf("        issuer:      %s\n", cert.Issuer.String())
		widgets.Printf("        sha256:      %s\n", fingerprint(cert))
		widgets.Printf("        expires:     %s\n", cert.NotAfter.Format(time.RFC3339))
	}

	widgets.Println("\nTrust-on-first-use: verify the sha256 fingerprint above with your IT/security team")
	widgets.Println("before relying on this bundle. Installing the CA into your OS trust store is safer.")

	if tlsFetchGlobal || tlsFetchSource != "" {
		if err := wireCABundle(outPath); err != nil {
			return err
		}
	} else {
		widgets.Printf("\nTo use it, set:  aide --ca-bundle %s ...\n", outPath)
		widgets.Println("or wire it into config with --source <name> or --global.")
	}
	return nil
}

func parseTLSTarget(raw string) (host, addr string) {
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	if i := strings.IndexByte(raw, '/'); i >= 0 {
		raw = raw[:i]
	}
	host = raw
	port := "443"
	if h, p, err := net.SplitHostPort(raw); err == nil {
		host, port = h, p
	}
	return host, net.JoinHostPort(host, port)
}

func resolveCertOutput(host string) (string, error) {
	if tlsFetchOutput != "" {
		return tlsFetchOutput, nil
	}
	dir := filepath.Join(xdg.AideHome(), "certs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}
	return filepath.Join(dir, host+".pem"), nil
}

func fingerprint(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.Raw)
	parts := make([]string, len(sum))
	for i, b := range sum {
		parts[i] = fmt.Sprintf("%02X", b)
	}
	return strings.Join(parts, ":")
}

func wireCABundle(path string) error {
	cfg, err := loadRawConfig()
	if err != nil {
		return err
	}

	if tlsFetchGlobal {
		cfg.Settings.TLS.CABundle = path
	}
	if tlsFetchSource != "" {
		src, exists := cfg.Sources[tlsFetchSource]
		if !exists {
			return fmt.Errorf("source '%s' not found", tlsFetchSource)
		}
		if src.TLS == nil {
			src.TLS = &config.TLS{}
		}
		src.TLS.CABundle = path
		cfg.Sources[tlsFetchSource] = src
	}

	if err := cfg.Save(cfgFile); err != nil {
		return err
	}

	switch {
	case tlsFetchGlobal && tlsFetchSource != "":
		widgets.Printf("\nWired into settings.tls.ca_bundle and sources.%s.tls.ca_bundle\n", tlsFetchSource)
	case tlsFetchGlobal:
		widgets.Println("\nWired into settings.tls.ca_bundle")
	default:
		widgets.Printf("\nWired into sources.%s.tls.ca_bundle\n", tlsFetchSource)
	}
	return nil
}
