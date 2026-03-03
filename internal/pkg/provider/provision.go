// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package provider implements hetzner infra provider core.
package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/siderolabs/omni/client/pkg/infra/provision"
	"github.com/siderolabs/omni/client/pkg/omni/resources/infra"
	"go.uber.org/zap"

	provconfig "github.com/theGunner295/omni-infra-provider-hetzner/internal/pkg/config"
	hclient "github.com/theGunner295/omni-infra-provider-hetzner/internal/pkg/hcloud"
	providermeta "github.com/theGunner295/omni-infra-provider-hetzner/internal/pkg/provider/meta"
	"github.com/theGunner295/omni-infra-provider-hetzner/internal/pkg/provider/resources"
)

// Provisioner implements the Hetzner infra provider.
type Provisioner struct {
	config  *provconfig.Config
	clients map[string]*hclient.Client // keyed by project_id
	logger  *zap.Logger
}

// NewProvisioner creates a new Hetzner Provisioner.
func NewProvisioner(cfg *provconfig.Config, logger *zap.Logger) *Provisioner {
	clients := make(map[string]*hclient.Client, len(cfg.Projects))

	for _, p := range cfg.Projects {
		clients[p.ProjectID] = hclient.NewClient(p.Token, logger)
	}

	return &Provisioner{
		config:  cfg,
		clients: clients,
		logger:  logger,
	}
}

// clientForProject returns the hcloud Client for the given project ID.
func (p *Provisioner) clientForProject(projectID string) (*hclient.Client, *provconfig.ProjectConfig, error) {
	proj, err := p.config.GetProject(projectID)
	if err != nil {
		return nil, nil, err
	}

	client, ok := p.clients[proj.ProjectID]
	if !ok {
		return nil, nil, fmt.Errorf("no client for project %q", proj.ProjectID)
	}

	return client, proj, nil
}

// ProvisionSteps implements provision.Provisioner.
func (p *Provisioner) ProvisionSteps() []provision.Step[*resources.Machine] {
	return []provision.Step[*resources.Machine]{
		provision.NewStep(
			"createServer",
			func(ctx context.Context, logger *zap.Logger, pctx provision.Context[*resources.Machine]) error {
				// If already provisioned, skip
				if pctx.State.TypedSpec().Value.ServerId != "" {
					logger.Info("server already created, skipping",
						zap.String("server_id", pctx.State.TypedSpec().Value.ServerId),
					)

					return nil
				}

				var data Data
				if err := pctx.UnmarshalProviderData(&data); err != nil {
					return fmt.Errorf("failed to unmarshal provider data: %w", err)
				}

				client, proj, err := p.clientForProject(data.ProjectID)
				if err != nil {
					return provision.NewRetryErrorf(time.Second*10, "failed to get project client: %w", err)
				}

				// Resolve location
				location := data.Location
				if location == "" {
					location = proj.DefaultLocation
				}

				if location == "" {
					return fmt.Errorf("location is required: set data.location or project default_location")
				}

				// Resolve snapshot name
				snapshotName := data.SnapshotName
				if snapshotName == "" {
					snapshotName = proj.SnapshotName
				}

				if snapshotName == "" {
					return fmt.Errorf("snapshot_name is required: set data.snapshot_name or project snapshot_name")
				}

				if data.ServerType == "" {
					return fmt.Errorf("server_type is required")
				}

				// Validate SKU + location combination
				if err := client.ValidateSKUAndLocation(ctx, data.ServerType, location); err != nil {
					return provision.NewRetryErrorf(time.Second*30, "SKU/location validation failed: %w", err)
				}

				// Find snapshot image
				image, err := client.FindSnapshotByName(ctx, snapshotName)
				if err != nil {
					return provision.NewRetryErrorf(time.Second*30, "failed to find snapshot %q: %w", snapshotName, err)
				}

				serverName := pctx.GetRequestID()

				createOpts := hcloud.ServerCreateOpts{
					Name:       serverName,
					ServerType: &hcloud.ServerType{Name: data.ServerType},
					Image:      image,
					Location:   &hcloud.Location{Name: location},
					Labels: map[string]string{
						"omni-provider": providermeta.ProviderID,
						"machine-id":    serverName,
					},
				}

				// Configure networking mode
				networkMode := data.ResolveNetworkMode()

				if networkMode == NetworkModePrivate {
					if data.NetworkName == "" {
						return fmt.Errorf("network_name is required when network_mode is %q", NetworkModePrivate)
					}

					network, err := client.FindNetworkByName(ctx, data.NetworkName)
					if err != nil {
						return provision.NewRetryErrorf(time.Second*30, "failed to find network %q: %w", data.NetworkName, err)
					}

					createOpts.Networks = []*hcloud.Network{network}
					createOpts.PublicNet = &hcloud.ServerCreatePublicNet{
						EnableIPv4: false,
						EnableIPv6: false,
					}
				} else {
					// Public IP mode: allocate public IPv4+IPv6
					createOpts.PublicNet = &hcloud.ServerCreatePublicNet{
						EnableIPv4: true,
						EnableIPv6: true,
					}
				}

				// Inject Talos join config via user data
				if pctx.ConnectionParams.JoinConfig != "" {
					createOpts.UserData = pctx.ConnectionParams.JoinConfig
				}

				logger.Info("creating Hetzner server",
					zap.String("name", serverName),
					zap.String("server_type", data.ServerType),
					zap.String("location", location),
					zap.String("snapshot", snapshotName),
					zap.String("network_mode", networkMode),
				)

				var result hcloud.ServerCreateResult

				err = client.Do(ctx, "CreateServer", func() error {
					var innerErr error

					result, _, innerErr = client.Inner().Server.Create(ctx, createOpts)

					return innerErr
				})
				if err != nil {
					return provision.NewRetryErrorf(time.Second*30, "failed to create server: %w", err)
				}

				serverID := strconv.FormatInt(result.Server.ID, 10)

				pctx.State.TypedSpec().Value.ServerId = serverID
				pctx.State.TypedSpec().Value.Location = location
				pctx.State.TypedSpec().Value.ProjectId = proj.ProjectID

				logger.Info("Hetzner server created",
					zap.String("server_id", serverID),
					zap.String("name", serverName),
				)

				return nil
			},
		),
		provision.NewStep(
			"waitForServerReady",
			func(ctx context.Context, logger *zap.Logger, pctx provision.Context[*resources.Machine]) error {
				serverIDStr := pctx.State.TypedSpec().Value.ServerId
				if serverIDStr == "" {
					return provision.NewRetryErrorf(time.Second*5, "waiting for server to be created")
				}

				client, _, err := p.clientForProject(pctx.State.TypedSpec().Value.ProjectId)
				if err != nil {
					return provision.NewRetryErrorf(time.Second*10, "failed to get project client: %w", err)
				}

				serverID, err := strconv.ParseInt(serverIDStr, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid server ID %q: %w", serverIDStr, err)
				}

				server, err := client.GetServerByID(ctx, serverID)
				if err != nil {
					return provision.NewRetryErrorf(time.Second*10, "failed to get server status: %w", err)
				}

				if server == nil {
					return provision.NewRetryErrorf(time.Second*10, "server not found, waiting")
				}

				if server.Status != hcloud.ServerStatusRunning {
					logger.Info("waiting for server to be running",
						zap.String("server_id", serverIDStr),
						zap.String("status", string(server.Status)),
					)

					return provision.NewRetryErrorf(time.Second*10, "server is not running yet (status: %s)", server.Status)
				}

				logger.Info("server is running",
					zap.String("server_id", serverIDStr),
				)

				return nil
			},
		),
	}
}

// Deprovision implements provision.Provisioner.
func (p *Provisioner) Deprovision(
	ctx context.Context,
	logger *zap.Logger,
	machine *resources.Machine,
	_ *infra.MachineRequest,
) error {
	serverIDStr := machine.TypedSpec().Value.ServerId
	if serverIDStr == "" {
		// Server was never created (failed early in provisioning)
		logger.Info("server ID not found in machine state, skipping deprovision")

		return nil
	}

	projectID := machine.TypedSpec().Value.ProjectId

	client, _, err := p.clientForProject(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project client for deprovision: %w", err)
	}

	serverID, err := strconv.ParseInt(serverIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid server ID %q: %w", serverIDStr, err)
	}

	logger.Info("deprovisioning Hetzner server",
		zap.String("server_id", serverIDStr),
		zap.String("project_id", projectID),
	)

	server, err := client.GetServerByID(ctx, serverID)
	if err != nil {
		return fmt.Errorf("failed to get server %s: %w", serverIDStr, err)
	}

	if server == nil {
		logger.Info("server not found, already deleted", zap.String("server_id", serverIDStr))

		return nil
	}

	// Power off the server if running
	if server.Status == hcloud.ServerStatusRunning {
		logger.Info("powering off server before deletion", zap.String("server_id", serverIDStr))

		var action *hcloud.Action

		err = client.Do(ctx, "PowerOffServer", func() error {
			var innerErr error

			action, _, innerErr = client.Inner().Server.Poweroff(ctx, server)

			return innerErr
		})
		if err != nil {
			return fmt.Errorf("failed to power off server %s: %w", serverIDStr, err)
		}

		// Wait for the power-off action to complete
		if err = waitForAction(ctx, client, action); err != nil {
			return fmt.Errorf("power-off action failed for server %s: %w", serverIDStr, err)
		}

		logger.Info("server powered off", zap.String("server_id", serverIDStr))
	}

	// Delete the server
	logger.Info("deleting server", zap.String("server_id", serverIDStr))

	err = client.Do(ctx, "DeleteServer", func() error {
		result, _, innerErr := client.Inner().Server.DeleteWithResult(ctx, server)
		if innerErr != nil {
			return innerErr
		}

		// Wait for delete action
		return waitForAction(ctx, client, result.Action)
	})

	var hcloudErr hcloud.Error
	if errors.As(err, &hcloudErr) && hcloudErr.Code == hcloud.ErrorCodeNotFound {
		logger.Info("server already deleted", zap.String("server_id", serverIDStr))

		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to delete server %s: %w", serverIDStr, err)
	}

	logger.Info("server deleted successfully", zap.String("server_id", serverIDStr))

	return nil
}

// waitForAction polls a Hetzner action until it completes or returns an error.
func waitForAction(ctx context.Context, client *hclient.Client, action *hcloud.Action) error {
	if action == nil {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}

		var updated *hcloud.Action

		err := client.Do(ctx, "GetAction", func() error {
			var innerErr error

			updated, _, innerErr = client.Inner().Action.GetByID(ctx, action.ID)

			return innerErr
		})
		if err != nil {
			return fmt.Errorf("failed to poll action %d: %w", action.ID, err)
		}

		if updated == nil {
			return nil
		}

		switch updated.Status {
		case hcloud.ActionStatusSuccess:
			return nil
		case hcloud.ActionStatusError:
			return fmt.Errorf("action %d failed: %s", updated.ID, updated.ErrorMessage)
		}
	}
}
