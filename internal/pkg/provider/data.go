// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package provider

const (
	// NetworkModePublic allocates and attaches a public IPv4/IPv6 address to the server.
	NetworkModePublic = "public"
	// NetworkModePrivate deploys VMs into an existing Hetzner private network without public IP.
	NetworkModePrivate = "private"
)

// Data is the provider custom machine config supplied per machine request via the Omni schema.
type Data struct {
	// ProjectID selects which configured Hetzner project to use.
	// If empty, defaults to the single configured project (error if multiple are configured).
	ProjectID string `yaml:"project_id"`

	// ServerType is the Hetzner server type/SKU (e.g. cx23, cx32, cx52).
	ServerType string `yaml:"server_type"`

	// Location is the Hetzner data center location (e.g. fsn1, nbg1, hel1).
	// Overrides the project default_location when set.
	Location string `yaml:"location"`

	// SnapshotName is the name of the snapshot to use for VM provisioning.
	// Overrides the project snapshot_name when set.
	SnapshotName string `yaml:"snapshot_name"`

	// NetworkMode controls how network is assigned. Valid values: "public" (default), "private".
	NetworkMode string `yaml:"network_mode"`

	// NetworkName is the name of the private Hetzner network to attach the server to.
	// Required when NetworkMode is "private".
	NetworkName string `yaml:"network_name"`
}

// ResolveNetworkMode returns the effective network mode, defaulting to public.
func (d *Data) ResolveNetworkMode() string {
	if d.NetworkMode == NetworkModePrivate {
		return NetworkModePrivate
	}

	return NetworkModePublic
}
