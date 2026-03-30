//go:build nxuskit

// Package main demonstrates the nxuskit authentication helper APIs.
//
// This example shows how to manage provider credentials using the SDK:
//   - List available providers and their auth methods
//   - Check authentication status per provider
//   - Set/remove API key credentials
//   - Initiate OAuth login flows (infrastructure ready, no providers use it yet)
//
// Usage:
//
//	go run .                       # Show auth dashboard
//	go run . --provider openai     # Check status for a specific provider
//	go run . --login openai        # Set credential interactively
//	go run . --oauth openai        # Start OAuth flow (if supported)
//	go run . --providers           # List all providers with auth info
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func main() {
	providerFlag := flag.String("provider", "", "Check auth status for a specific provider")
	loginFlag := flag.String("login", "", "Set API key credential for a provider (reads from env)")
	oauthFlag := flag.String("oauth", "", "Start OAuth flow for a provider")
	providersFlag := flag.Bool("providers", false, "List all providers with auth metadata")
	flag.Parse()

	switch {
	case *providersFlag:
		listProviders()
	case *providerFlag != "":
		checkProvider(*providerFlag)
	case *loginFlag != "":
		setCredential(*loginFlag)
	case *oauthFlag != "":
		startOAuth(*oauthFlag)
	default:
		showDashboard()
	}
}

// listProviders shows all available providers with their auth methods.
func listProviders() {
	fmt.Println("=== nxuskit Provider Auth Registry ===")
	fmt.Println()

	providers := nxuskit.AuthProviders()
	if len(providers) == 0 {
		fmt.Println("No providers registered.")
		return
	}

	fmt.Printf("%-20s %-12s %-8s %s\n", "Provider", "Auth Method", "OAuth", "Env Variable")
	fmt.Println(strings.Repeat("-", 72))

	for _, p := range providers {
		oauthStr := "no"
		if p.OAuthCapable {
			oauthStr = "yes"
		}
		authMethods := strings.Join(p.AuthMethods, ", ")
		if authMethods == "" {
			authMethods = "—"
		}
		fmt.Printf("%-20s %-12s %-8s %s\n",
			p.ProviderID, authMethods, oauthStr, p.EnvVarName)
	}

	fmt.Printf("\n%d providers total\n", len(providers))
}

// checkProvider shows auth status for a specific provider.
func checkProvider(providerID string) {
	fmt.Printf("=== Auth Status: %s ===\n\n", providerID)

	status, err := nxuskit.GetAuthStatus(providerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	printStatus(status)

	// Also show OAuth status if the provider supports it
	oauthStatus, err := nxuskit.OAuthGetStatus(providerID)
	if err == nil {
		fmt.Println("\nOAuth Status:")
		if oauthStatus.Authenticated {
			fmt.Println("  Authenticated: yes")
			if oauthStatus.ExpiresAt != nil {
				t := time.Unix(*oauthStatus.ExpiresAt, 0)
				fmt.Printf("  Expires: %s\n", t.Format(time.RFC3339))
			}
			if len(oauthStatus.Scopes) > 0 {
				fmt.Printf("  Scopes: %s\n", strings.Join(oauthStatus.Scopes, ", "))
			}
		} else {
			fmt.Println("  Authenticated: no")
		}
	}
}

// setCredential sets an API key for a provider from an environment variable.
func setCredential(providerID string) {
	// Find the provider's env var
	providers := nxuskit.AuthProviders()
	var envVar string
	for _, p := range providers {
		if p.ProviderID == providerID {
			envVar = p.EnvVarName
			break
		}
	}

	if envVar == "" {
		fmt.Fprintf(os.Stderr, "Unknown provider: %s\n", providerID)
		os.Exit(1)
	}

	apiKey := os.Getenv(envVar)
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "Set %s environment variable first, then run:\n", envVar)
		fmt.Fprintf(os.Stderr, "  %s=%s go run . --login %s\n", envVar, "<your-key>", providerID)
		os.Exit(1)
	}

	if err := nxuskit.AuthSetCredential(providerID, apiKey); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting credential: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Credential stored for %s\n", providerID)

	// Verify
	status, err := nxuskit.GetAuthStatus(providerID)
	if err == nil {
		printStatus(status)
	}
}

// startOAuth initiates an OAuth flow for a provider.
func startOAuth(providerID string) {
	fmt.Printf("Starting OAuth flow for %s...\n", providerID)
	fmt.Println("A browser window will open for authentication.")
	fmt.Println("Waiting for callback (timeout: 120s)...")
	fmt.Println()

	result, err := nxuskit.OAuthStart(providerID, 120)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OAuth error: %v\n", err)
		os.Exit(1)
	}

	if result.Success {
		fmt.Printf("OAuth authentication successful for %s!\n", result.ProviderID)
		fmt.Printf("Message: %s\n", result.Message)
	} else {
		fmt.Printf("OAuth authentication failed for %s\n", result.ProviderID)
		if result.Error != nil {
			fmt.Printf("Error: %s\n", *result.Error)
		}
		os.Exit(1)
	}
}

// showDashboard displays auth status for all providers.
func showDashboard() {
	fmt.Println("=== nxuskit Auth Dashboard ===")
	fmt.Println()

	statuses, err := nxuskit.AuthStatusAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(statuses) == 0 {
		fmt.Println("No providers configured.")
		return
	}

	authenticated := 0
	for _, s := range statuses {
		printStatus(&s)
		fmt.Println()
		if authStatusReady(s.Status) {
			authenticated++
		}
	}

	fmt.Printf("--- %d/%d providers authenticated ---\n", authenticated, len(statuses))
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  --providers         List all available providers")
	fmt.Println("  --provider <name>   Check specific provider status")
	fmt.Println("  --login <name>      Set credential from env variable")
	fmt.Println("  --oauth <name>      Start OAuth flow (if supported)")
}

// printStatus prints a formatted AuthStatus.
func printStatus(s *nxuskit.AuthStatus) {
	meta, hasMeta := providerAuthMeta(s.ProviderID)
	fmt.Printf("  Provider: %s\n", s.ProviderID)
	fmt.Printf("  Status:   %s\n", authStatusLabel(s.Status))
	src := s.Source
	if src == "" {
		src = "—"
	}
	fmt.Printf("  Source:   %s\n", src)
	if s.MaskedPreview != "" {
		fmt.Printf("  Preview:  %s\n", s.MaskedPreview)
	}
	if hasMeta && meta.OAuthCapable {
		fmt.Println("  OAuth:    supported")
	}
	if hasMeta && len(meta.AuthMethods) > 0 {
		fmt.Printf("  Methods:  %s\n", strings.Join(meta.AuthMethods, ", "))
	}
}

func authStatusReady(status string) bool {
	switch status {
	case "authenticated_env", "authenticated_store", "not_required":
		return true
	default:
		return false
	}
}

func authStatusLabel(status string) string {
	switch status {
	case "not_required":
		return "not required"
	case "authenticated_env", "authenticated_store":
		return "authenticated"
	case "not_authenticated":
		return "not authenticated"
	default:
		return status
	}
}

func providerAuthMeta(providerID string) (nxuskit.ProviderAuthMetadata, bool) {
	for _, p := range nxuskit.AuthProviders() {
		if p.ProviderID == providerID {
			return p, true
		}
	}
	return nxuskit.ProviderAuthMetadata{}, false
}
