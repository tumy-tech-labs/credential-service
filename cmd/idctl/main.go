package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bradtumy/credential-service/internal/did"
	sdk "github.com/bradtumy/credential-service/sdk/go"
)

const (
	issuerURLEnv   = "CRED_ISSUER_URL"
	verifierURLEnv = "CRED_VERIFIER_URL"

	defaultIssuerURL   = "http://localhost:8080"
	defaultVerifierURL = "http://localhost:8081"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "did":
		err = handleDID(os.Args[2:])
	case "vc":
		err = handleVC(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func handleDID(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "missing did subcommand")
		didUsage()
		return errors.New("missing did subcommand")
	}

	switch args[0] {
	case "create":
		return didCreate(args[1:])
	case "export":
		return didExport(args[1:])
	case "import":
		return didImport(args[1:])
	case "rotate":
		return didRotate(args[1:])
	case "-h", "--help", "help":
		didUsage()
		return nil
	default:
		fmt.Fprintf(os.Stderr, "unknown did command: %s\n", args[0])
		didUsage()
		return fmt.Errorf("unknown did command: %s", args[0])
	}
}

func didCreate(args []string) error {
	fs := flag.NewFlagSet("did create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	algorithm := fs.String("alg", "EdDSA", "Key algorithm (EdDSA or ES256)")
	storePath := fs.String("store", did.DefaultStorePath(), "Path to DID store JSON file")
	label := fs.String("label", "", "Optional label for this DID (defaults to DID value)")

	if err := fs.Parse(args); err != nil {
		fs.SetOutput(os.Stderr)
		fs.Usage()
		return err
	}

	doc, err := did.NewDIDJWKWithAlg(*algorithm)
	if err != nil {
		return fmt.Errorf("generate did:jwk: %w", err)
	}

	store := did.NewFileStore(*storePath)
	if err := store.Load(); err != nil {
		return fmt.Errorf("load DID store: %w", err)
	}

	if err := store.Save(*label, doc); err != nil {
		return fmt.Errorf("save DID: %w", err)
	}

	return printDID(doc)
}

func didExport(args []string) error {
	fs := flag.NewFlagSet("did export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	label := fs.String("label", "", "Label of the DID to export")
	output := fs.String("output", "", "File to write the DID document (default: stdout)")
	storePath := fs.String("store", did.DefaultStorePath(), "Path to DID store JSON file")

	if err := fs.Parse(args); err != nil {
		fs.SetOutput(os.Stderr)
		fs.Usage()
		return err
	}

	if *label == "" {
		return errors.New("--label is required")
	}

	store, err := loadStore(*storePath)
	if err != nil {
		return err
	}

	doc, ok := store.Get(*label)
	if !ok {
		return fmt.Errorf("did not find entry for label %s", *label)
	}

	data, err := did.ExportDID(doc)
	if err != nil {
		return fmt.Errorf("export DID: %w", err)
	}

	if *output == "" {
		os.Stdout.Write(data)
		if len(data) == 0 || data[len(data)-1] != '\n' {
			fmt.Fprintln(os.Stdout)
		}
		return nil
	}

	return os.WriteFile(*output, data, 0o600)
}

func didImport(args []string) error {
	fs := flag.NewFlagSet("did import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	file := fs.String("file", "", "Path to a DID document JSON file")
	label := fs.String("label", "", "Label to store the DID under (default: DID value)")
	storePath := fs.String("store", did.DefaultStorePath(), "Path to DID store JSON file")

	if err := fs.Parse(args); err != nil {
		fs.SetOutput(os.Stderr)
		fs.Usage()
		return err
	}

	if *file == "" {
		return errors.New("--file is required")
	}

	data, err := os.ReadFile(*file)
	if err != nil {
		return fmt.Errorf("read DID file: %w", err)
	}

	doc, err := did.ImportDID(data)
	if err != nil {
		return fmt.Errorf("import DID: %w", err)
	}

	store, err := loadStore(*storePath)
	if err != nil {
		return err
	}

	if err := store.Save(*label, doc); err != nil {
		return fmt.Errorf("save DID: %w", err)
	}

	return printDID(doc)
}

func didRotate(args []string) error {
	fs := flag.NewFlagSet("did rotate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	label := fs.String("label", "", "Label of the DID to rotate")
	storePath := fs.String("store", did.DefaultStorePath(), "Path to DID store JSON file")

	if err := fs.Parse(args); err != nil {
		fs.SetOutput(os.Stderr)
		fs.Usage()
		return err
	}

	if *label == "" {
		return errors.New("--label is required")
	}

	store, err := loadStore(*storePath)
	if err != nil {
		return err
	}

	doc, err := store.Rotate(*label)
	if err != nil {
		return fmt.Errorf("rotate DID: %w", err)
	}

	return printDID(doc)
}

func handleVC(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "missing vc subcommand")
		vcUsage()
		return errors.New("missing vc subcommand")
	}

	switch args[0] {
	case "issue":
		return vcIssue(args[1:])
	case "verify":
		return vcVerify(args[1:])
	case "-h", "--help", "help":
		vcUsage()
		return nil
	default:
		fmt.Fprintf(os.Stderr, "unknown vc command: %s\n", args[0])
		vcUsage()
		return fmt.Errorf("unknown vc command: %s", args[0])
	}
}

func vcIssue(args []string) error {
	fs := flag.NewFlagSet("vc issue", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	subject := fs.String("subject-did", "", "Subject DID for the credential (required)")
	scopeStr := fs.String("scope", "", "Comma-separated scope values")
	ttl := fs.Int64("ttl-seconds", 300, "Credential TTL in seconds")
	issuerURL := fs.String("issuer-url", envDefault(issuerURLEnv, defaultIssuerURL), "Issuer service URL")

	if err := fs.Parse(args); err != nil {
		fs.SetOutput(os.Stderr)
		fs.Usage()
		return err
	}

	if *subject == "" {
		return errors.New("--subject-did is required")
	}

	claims := map[string]interface{}{}
	scope := parseScope(*scopeStr)
	if len(scope) > 0 {
		claims["scope"] = scope
	}

	client := sdk.Client{IssuerURL: *issuerURL}
	resp, err := client.IssueCredential(context.Background(), sdk.IssueRequest{
		SubjectDID: *subject,
		TTLSeconds: *ttl,
		Claims:     claims,
	})
	if err != nil {
		return fmt.Errorf("issue credential: %w", err)
	}

	fmt.Fprintln(os.Stdout, resp.Credential)
	return nil
}

func vcVerify(args []string) error {
	fs := flag.NewFlagSet("vc verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	credential := fs.String("credential", "", "Credential string to verify")
	credentialFile := fs.String("credential-file", "", "Path to file containing the credential")
	verifierURL := fs.String("verifier-url", envDefault(verifierURLEnv, defaultVerifierURL), "Verifier service URL")

	if err := fs.Parse(args); err != nil {
		fs.SetOutput(os.Stderr)
		fs.Usage()
		return err
	}

	token := strings.TrimSpace(*credential)
	if token == "" && *credentialFile != "" {
		data, err := os.ReadFile(*credentialFile)
		if err != nil {
			return fmt.Errorf("read credential file: %w", err)
		}
		token = strings.TrimSpace(string(data))
	}

	if token == "" {
		return errors.New("either --credential or --credential-file must be provided")
	}

	client := sdk.Client{VerifierURL: *verifierURL}
	resp, err := client.Verify(context.Background(), token)
	if err != nil {
		return fmt.Errorf("verify credential: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

func usage() {
	fmt.Fprintf(os.Stderr, `idctl - credential service CLI

Usage:
  idctl did <command> [options]
  idctl vc <command> [options]

Commands:
  did create           Generate a did:jwk identifier and save to the local store
  did export           Export a stored DID document
  did import           Import a DID document JSON file
  did rotate           Rotate a DID's keys and update the store
  vc issue             Issue a verifiable credential
  vc verify            Verify a credential
`)
}

func didUsage() {
	fmt.Fprintf(os.Stderr, `Usage: idctl did <command> [options]

Commands:
  create   Generate a did:jwk identifier (options: --alg, --store, --label)
  export   Export a DID document (options: --label, --output, --store)
  import   Import a DID document (options: --file, --label, --store)
  rotate   Rotate a DID's key (options: --label, --store)
`)
}

func vcUsage() {
	fmt.Fprintf(os.Stderr, `Usage: idctl vc <command> [options]

Commands:
  issue   Issue a verifiable credential
  verify  Verify a credential
`)
}

func envDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func loadStore(path string) (*did.FileStore, error) {
	store := did.NewFileStore(path)
	if err := store.Load(); err != nil {
		return nil, fmt.Errorf("load DID store: %w", err)
	}
	return store, nil
}

func printDID(doc did.DIDDocument) error {
	key, err := doc.CurrentKey()
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"did":             doc.DID,
		"algorithm":       key.Algorithm,
		"current_key_id":  doc.CurrentKeyID,
		"public_jwk":      key.PublicJWK,
		"private_key_pem": strings.TrimSpace(key.PrivateKeyPEM),
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func parseScope(scope string) []string {
	if scope == "" {
		return nil
	}
	parts := strings.Split(scope, ",")
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return cleaned
}
