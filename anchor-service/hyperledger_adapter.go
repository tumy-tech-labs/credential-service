package main

import (
	"fmt"
	"os"

	"github.com/cloudflare/cfssl/log"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

// HyperledgerAdapter struct definition
type HyperledgerAdapter struct {
	sdk    *fabsdk.FabricSDK
	client *channel.Client
}

type HyperledgerConfig struct {
	ChannelID      string
	User           string
	ChaincodeID    string
	PeerAddress    string
	LocalMSPID     string
	OrdererAddress string
	OrdererMSPID   string
}

// NewHyperledgerAdapter initializes the adapter with the Fabric SDK
// NewHyperledgerAdapter initializes the Hyperledger SDK using the YAML configuration
func NewHyperledgerAdapter() (*HyperledgerAdapter, error) {

	log.Debug("Creating a New Client to the Hyperledger Fabric")

	// Path to the configuration file
	configPath := "/etc/hyperledger/fabric/config.yaml" // Ensure this path matches your Docker setup

	// Initialize the Fabric SDK with the provided YAML configuration
	sdk, err := fabsdk.New(config.FromFile(configPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create new Fabric SDK: %v", err)
	}

	log.Debug("I got the client connected: ", sdk)

	// Define the channel context using the user specified in the YAML configuration
	clientContext := sdk.ChannelContext("mychannel", fabsdk.WithUser("Admin"))
	client, err := channel.New(clientContext)
	if err != nil {
		return nil, fmt.Errorf("failed to create new Fabric channel client: %v", err)
	}

	return &HyperledgerAdapter{
		sdk:    sdk,
		client: client,
	}, nil
}

// AnchorDID anchors a DID to the Hyperledger Fabric blockchain
func (h *HyperledgerAdapter) AnchorDID(did string) (string, error) {
	if h.client == nil {
		return "", fmt.Errorf("hyperledger client is not initialized")
	}

	// Prepare the arguments to call the chaincode function for anchoring the DID
	args := [][]byte{
		[]byte("AnchorDID"), // Function name in the chaincode
		[]byte(did),         // DID to be anchored
	}

	// Invoke the chaincode on Hyperledger Fabric
	response, err := h.client.Execute(channel.Request{
		ChaincodeID: os.Getenv("FABRIC_CHAINCODE_ID"), // Ensure this variable is set in your environment
		Fcn:         "AnchorDID",
		Args:        args,
	})
	if err != nil {
		return "", fmt.Errorf("failed to anchor DID: %v", err)
	}

	// Return the transaction ID as the result
	return string(response.TransactionID), nil
}

// AnchorCredential anchors a Verifiable Credential to the blockchain
func (h *HyperledgerAdapter) AnchorCredential(credential string) (string, error) {
	if h.client == nil {
		return "", fmt.Errorf("Hyperledger client is not initialized")
	}

	// Prepare the arguments to call the chaincode function for anchoring the credential
	args := [][]byte{
		[]byte("AnchorCredential"), // Function name in the chaincode
		[]byte(credential),         // Credential data to be anchored
	}

	// Invoke the chaincode on Hyperledger Fabric
	response, err := h.client.Execute(channel.Request{
		ChaincodeID: os.Getenv("FABRIC_CHAINCODE_ID"), // Ensure this variable is set in your environment
		Fcn:         "AnchorCredential",
		Args:        args,
	})
	if err != nil {
		return "", fmt.Errorf("failed to anchor credential: %v", err)
	}

	// Return the transaction ID as the result
	return string(response.TransactionID), nil
}

// Close shuts down the HyperledgerAdapter and releases resources
func (h *HyperledgerAdapter) Close() error {
	if h.sdk != nil {
		h.sdk.Close()
	}
	return nil
}
