package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/bradtumy/credential-service/internal/did"
)

func main() {
	algorithm := flag.String("algorithm", "EdDSA", "Signing algorithm: EdDSA (Ed25519) or ES256 (ECDSA P-256)")
	output := flag.String("output", "", "Optional: Output file path for private key PEM (e.g., alice-key.pem)")
	didOnly := flag.Bool("did-only", false, "Only output the DID, not the private key")
	help := flag.Bool("help", false, "Show usage information")

	flag.Parse()

	if *help {
		printUsage()
		os.Exit(0)
	}

	// Validate algorithm
	if *algorithm != "EdDSA" && *algorithm != "ES256" {
		fmt.Fprintf(os.Stderr, "Error: Unsupported algorithm '%s'. Supported: EdDSA, ES256\n", *algorithm)
		os.Exit(1)
	}

	// Generate the DID and key pair
	doc, err := did.NewDIDJWKWithAlg(*algorithm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating DID: %v\n", err)
		os.Exit(1)
	}

	key, err := doc.CurrentKey()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting current key: %v\n", err)
		os.Exit(1)
	}

	publicJWK, err := json.Marshal(key.PublicJWK)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding public JWK: %v\n", err)
		os.Exit(1)
	}

	// Always output the DID
	fmt.Printf("DID: %s\n", doc.DID)
	fmt.Printf("Algorithm: %s\n", key.Algorithm)
	fmt.Printf("Public JWK: %s\n", string(publicJWK))

	// Output private key if requested
	if !*didOnly {
		if *output != "" {
			// Save private key to file
			err := os.WriteFile(*output, []byte(key.PrivateKeyPEM), 0600)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing private key to file: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("\n✓ Private key saved to: %s (permissions: 0600)\n", *output)
			fmt.Println("\nKeep this file secure! It can be used to sign credentials as this DID.")
		} else {
			// Output private key to stdout
			fmt.Printf("\nPrivate Key (PEM format):\n%s\n", key.PrivateKeyPEM)
			fmt.Println("⚠️  Keep this private key secure! It can be used to sign credentials as this DID.")
		}
	}

	fmt.Println("\nYou can now use this DID in credential issuance requests:")
	fmt.Printf("  subject_did: \"%s\"\n", doc.DID)
}

func printUsage() {
	fmt.Println("keygen - Generate decentralized identifiers (DIDs) with embedded keys")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  keygen [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -algorithm string")
	fmt.Println("        Signing algorithm: EdDSA (Ed25519) or ES256 (ECDSA P-256) (default \"EdDSA\")")
	fmt.Println("  -output string")
	fmt.Println("        Output file path for private key PEM (e.g., alice-key.pem)")
	fmt.Println("  -did-only")
	fmt.Println("        Only output the DID, not the private key")
	fmt.Println("  -help")
	fmt.Println("        Show this usage information")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Generate EdDSA (Ed25519) key and save private key to file")
	fmt.Println("  keygen -output alice-key.pem")
	fmt.Println()
	fmt.Println("  # Generate ES256 (ECDSA P-256) key")
	fmt.Println("  keygen -algorithm ES256 -output bob-key.pem")
	fmt.Println()
	fmt.Println("  # Generate key and print to stdout")
	fmt.Println("  keygen")
	fmt.Println()
	fmt.Println("  # Generate only the DID (no private key)")
	fmt.Println("  keygen -did-only")
	fmt.Println()
	fmt.Println("Security:")
	fmt.Println("  - Private keys are saved with 0600 permissions (owner read/write only)")
	fmt.Println("  - Never share or commit private keys to version control")
	fmt.Println("  - The DID contains the public key and can be shared freely")
}
