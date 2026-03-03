// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package provider_test

import (
	"testing"

	"github.com/theGunner295/omni-infra-provider-hetzner/internal/pkg/provider"
)

func TestDataResolveNetworkMode(t *testing.T) {
	tests := []struct {
		name     string
		data     provider.Data
		expected string
	}{
		{
			name:     "empty defaults to public",
			data:     provider.Data{},
			expected: provider.NetworkModePublic,
		},
		{
			name:     "explicit public",
			data:     provider.Data{NetworkMode: provider.NetworkModePublic},
			expected: provider.NetworkModePublic,
		},
		{
			name:     "private mode",
			data:     provider.Data{NetworkMode: provider.NetworkModePrivate},
			expected: provider.NetworkModePrivate,
		},
		{
			name:     "unknown value defaults to public",
			data:     provider.Data{NetworkMode: "invalid"},
			expected: provider.NetworkModePublic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode := tt.data.ResolveNetworkMode()
			if mode != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, mode)
			}
		})
	}
}

func TestDataFields(t *testing.T) {
	d := provider.Data{
		ProjectID:    "proj-1",
		ServerType:   "cx32",
		Location:     "fsn1",
		SnapshotName: "talos-v1.9",
		NetworkMode:  provider.NetworkModePrivate,
		NetworkName:  "my-network",
	}

	if d.ProjectID != "proj-1" {
		t.Errorf("ProjectID mismatch")
	}

	if d.ServerType != "cx32" {
		t.Errorf("ServerType mismatch")
	}

	if d.Location != "fsn1" {
		t.Errorf("Location mismatch")
	}

	if d.SnapshotName != "talos-v1.9" {
		t.Errorf("SnapshotName mismatch")
	}

	if d.ResolveNetworkMode() != provider.NetworkModePrivate {
		t.Errorf("expected private network mode")
	}

	if d.NetworkName != "my-network" {
		t.Errorf("NetworkName mismatch")
	}
}
