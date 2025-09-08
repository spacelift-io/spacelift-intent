// Copyright 2025 Spacelift, Inc. and contributors
// SPDX-License-Identifier: Apache-2.0

//go:build legacy_plugin

package legacy

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	pb "github.com/apparentlymart/opentofu-providers/tofuprovider/grpc/tfplugin5"
)

// grpcProviderPlugin implements plugin.GRPCPlugin for the provider
type grpcProviderPlugin struct {
	plugin.Plugin
}

func (p *grpcProviderPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (any, error) {
	return pb.NewProviderClient(c), nil
}

func (p *grpcProviderPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	return fmt.Errorf("server not supported")
}

// providerInfo holds internal provider information including plugin client
type providerInfo struct {
	pluginClient *plugin.Client
	provider     pb.ProviderClient
	schema       *pb.GetProviderSchema_Response
	binary       string
	version      string
}

// Kill cleans up the plugin client
func (p *providerInfo) Kill() {
	if p.pluginClient != nil {
		p.pluginClient.Kill()
	}
}

// Implementation using hashicorp/go-plugin (only compiled with legacy_plugin build tag)
func startProviderPlugin(binary, providerName string) (*providerInfo, error) {
	// Create logger
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   fmt.Sprintf("terraform-provider-%s", providerName),
		Level:  hclog.Error,
		Output: os.Stderr,
	})

	// Create plugin client
	cmd := exec.Command(binary)
	cmd.Env = os.Environ()

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  5,
			MagicCookieKey:   "TF_PLUGIN_MAGIC_COOKIE",
			MagicCookieValue: "d602bf8f470bc67ca7faa0386276bbdd4330efaf76d1a219cb4d6991ca9872b2",
		},
		Plugins: map[string]plugin.Plugin{
			"provider": &grpcProviderPlugin{},
		},
		Cmd:              cmd,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger:           logger,
		SyncStdout:       os.Stdout,
		SyncStderr:       os.Stderr,
	})

	// Connect to plugin - kill client on any error after this point
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	// Get provider client
	raw, err := rpcClient.Dispense("provider")
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to dispense provider: %w", err)
	}

	provider, ok := raw.(pb.ProviderClient)
	if !ok {
		client.Kill()
		return nil, fmt.Errorf("provider is not a ProviderClient")
	}

	return &providerInfo{
		pluginClient: client,
		provider:     provider,
		binary:       binary,
	}, nil
}

// Cleanup function for hashicorp plugin
func cleanupPlugins() {
	plugin.CleanupClients()
}
