// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package config_test

import (
	"testing"

	"github.com/theGunner295/omni-infra-provider-hetzner/internal/pkg/config"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  config.Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty config",
			config:  config.Config{},
			wantErr: true,
			errMsg:  "at least one project must be configured",
		},
		{
			name: "missing token",
			config: config.Config{
				Projects: []config.ProjectConfig{
					{ProjectID: "proj1", SnapshotName: "snap1"},
				},
			},
			wantErr: true,
			errMsg:  "projects[0].token is required",
		},
		{
			name: "missing project_id",
			config: config.Config{
				Projects: []config.ProjectConfig{
					{Token: "tok1", SnapshotName: "snap1"},
				},
			},
			wantErr: true,
			errMsg:  "projects[0].project_id is required",
		},
		{
			name: "missing snapshot_name",
			config: config.Config{
				Projects: []config.ProjectConfig{
					{Token: "tok1", ProjectID: "proj1"},
				},
			},
			wantErr: true,
			errMsg:  "projects[0].snapshot_name is required",
		},
		{
			name: "duplicate project_id",
			config: config.Config{
				Projects: []config.ProjectConfig{
					{Token: "tok1", ProjectID: "proj1", SnapshotName: "snap1"},
					{Token: "tok2", ProjectID: "proj1", SnapshotName: "snap2"},
				},
			},
			wantErr: true,
			errMsg:  `projects[1].project_id "proj1" is duplicated`,
		},
		{
			name: "valid single project",
			config: config.Config{
				Projects: []config.ProjectConfig{
					{Token: "tok1", ProjectID: "proj1", SnapshotName: "snap1", DefaultLocation: "fsn1"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid multiple projects",
			config: config.Config{
				Projects: []config.ProjectConfig{
					{Token: "tok1", ProjectID: "proj1", SnapshotName: "snap1"},
					{Token: "tok2", ProjectID: "proj2", SnapshotName: "snap2"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.errMsg)
				}

				if err.Error() != tt.errMsg {
					t.Fatalf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %q", err.Error())
				}
			}
		})
	}
}

func TestConfigGetProject(t *testing.T) {
	cfg := config.Config{
		Projects: []config.ProjectConfig{
			{Token: "tok1", ProjectID: "proj-alpha", SnapshotName: "snap-alpha", DefaultLocation: "fsn1"},
			{Token: "tok2", ProjectID: "proj-beta", SnapshotName: "snap-beta", DefaultLocation: "nbg1"},
		},
	}

	t.Run("get by explicit project_id", func(t *testing.T) {
		proj, err := cfg.GetProject("proj-alpha")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if proj.Token != "tok1" {
			t.Fatalf("expected token tok1, got %q", proj.Token)
		}
	})

	t.Run("get second project by id", func(t *testing.T) {
		proj, err := cfg.GetProject("proj-beta")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if proj.DefaultLocation != "nbg1" {
			t.Fatalf("expected location nbg1, got %q", proj.DefaultLocation)
		}
	})

	t.Run("empty project_id with multiple projects returns error", func(t *testing.T) {
		_, err := cfg.GetProject("")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("non-existent project_id returns error", func(t *testing.T) {
		_, err := cfg.GetProject("proj-gamma")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty project_id with single project returns that project", func(t *testing.T) {
		singleCfg := config.Config{
			Projects: []config.ProjectConfig{
				{Token: "tok1", ProjectID: "only-project", SnapshotName: "snap1"},
			},
		}
		proj, err := singleCfg.GetProject("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if proj.ProjectID != "only-project" {
			t.Fatalf("expected only-project, got %q", proj.ProjectID)
		}
	})
}
