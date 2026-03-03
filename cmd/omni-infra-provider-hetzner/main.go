// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package main is the root cmd of the Hetzner infra provider.
package main

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/siderolabs/omni/client/pkg/client"
	"github.com/siderolabs/omni/client/pkg/infra"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"

	"github.com/theGunner295/omni-infra-provider-hetzner/internal/pkg/config"
	"github.com/theGunner295/omni-infra-provider-hetzner/internal/pkg/provider"
	"github.com/theGunner295/omni-infra-provider-hetzner/internal/pkg/provider/meta"
)

//go:embed data/schema.json
var schema string

//go:embed data/icon.svg
var icon []byte

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:          "omni-infra-provider-hetzner",
	Short:        "Hetzner Omni infrastructure provider",
	Long:         `Connects to Omni as an infra provider and manages VMs in Hetzner Cloud`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		loggerConfig := zap.NewProductionConfig()

		logger, err := loggerConfig.Build(
			zap.AddStacktrace(zapcore.ErrorLevel),
		)
		if err != nil {
			return fmt.Errorf("failed to create logger: %w", err)
		}

		if cfg.omniAPIEndpoint == "" {
			return fmt.Errorf("omni-api-endpoint flag is not set")
		}

		if cfg.configFile == "" {
			return fmt.Errorf("config-file flag is not set")
		}

		var hetznerConfig config.Config

		configFile, err := os.Open(cfg.configFile)
		if err != nil {
			return fmt.Errorf("failed to open config file %q: %w", cfg.configFile, err)
		}

		defer configFile.Close() //nolint:errcheck

		decoder := yaml.NewDecoder(configFile)

		if err = decoder.Decode(&hetznerConfig); err != nil {
			return fmt.Errorf("failed to parse config file %q: %w", cfg.configFile, err)
		}

		if err = hetznerConfig.Validate(); err != nil {
			return fmt.Errorf("invalid Hetzner configuration: %w", err)
		}

		provisioner := provider.NewProvisioner(&hetznerConfig, logger)

		ip, err := infra.NewProvider(meta.ProviderID, provisioner, infra.ProviderConfig{
			Name:        cfg.providerName,
			Description: cfg.providerDescription,
			Icon:        base64.RawStdEncoding.EncodeToString(icon),
			Schema:      schema,
		})
		if err != nil {
			return fmt.Errorf("failed to create infra provider: %w", err)
		}

		logger.Info("starting Hetzner infra provider")

		clientOptions := []client.Option{
			client.WithInsecureSkipTLSVerify(cfg.insecureSkipVerify),
		}

		if cfg.serviceAccountKey != "" {
			clientOptions = append(clientOptions, client.WithServiceAccount(cfg.serviceAccountKey))
		}

		return ip.Run(
			cmd.Context(),
			logger,
			infra.WithOmniEndpoint(cfg.omniAPIEndpoint),
			infra.WithClientOptions(clientOptions...),
			infra.WithEncodeRequestIDsIntoTokens(),
		)
	},
}

var cfg struct {
	omniAPIEndpoint     string
	serviceAccountKey   string
	providerName        string
	providerDescription string
	configFile          string
	insecureSkipVerify  bool
}

func main() {
	if err := app(); err != nil {
		os.Exit(1)
	}
}

func app() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	defer cancel()

	return rootCmd.ExecuteContext(ctx)
}

func init() {
	rootCmd.Flags().StringVar(&cfg.omniAPIEndpoint, "omni-api-endpoint", os.Getenv("OMNI_ENDPOINT"),
		"the endpoint of the Omni API, if not set, defaults to OMNI_ENDPOINT env var.")
	rootCmd.Flags().StringVar(&meta.ProviderID, "id", meta.ProviderID, "the id of the infra provider, used to match resources with the infra provider label.")
	rootCmd.Flags().StringVar(&cfg.serviceAccountKey, "omni-service-account-key", os.Getenv("OMNI_SERVICE_ACCOUNT_KEY"),
		"Omni service account key, if not set, defaults to OMNI_SERVICE_ACCOUNT_KEY.")
	rootCmd.Flags().StringVar(&cfg.providerName, "provider-name", "hetzner", "provider name as it appears in Omni")
	rootCmd.Flags().StringVar(&cfg.providerDescription, "provider-description", "Hetzner Cloud infrastructure provider", "Provider description as it appears in Omni")
	rootCmd.Flags().BoolVar(&cfg.insecureSkipVerify, "insecure-skip-verify", false, "ignores untrusted certs on Omni side")
	rootCmd.Flags().StringVar(&cfg.configFile, "config-file", "", "path to the Hetzner provider configuration file")
}
