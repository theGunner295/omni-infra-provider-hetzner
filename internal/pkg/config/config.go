// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package config describes Hetzner provider configuration.
package config

import "fmt"

// Config describes the Hetzner infra provider configuration.
type Config struct {
	Projects []ProjectConfig `yaml:"projects"`
}

// ProjectConfig describes a single Hetzner Cloud project.
type ProjectConfig struct {
	// Token is the Hetzner Cloud API token for this project.
	Token string `yaml:"token"`
	// ProjectID is the human-readable identifier for this project (used for selection).
	ProjectID string `yaml:"project_id"`
	// SnapshotName is the default snapshot name to use for VM provisioning.
	SnapshotName string `yaml:"snapshot_name"`
	// DefaultLocation is the default Hetzner location (e.g. fsn1, nbg1, hel1).
	DefaultLocation string `yaml:"default_location"`
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if len(c.Projects) == 0 {
		return fmt.Errorf("at least one project must be configured")
	}

	seen := make(map[string]struct{})

	for i, p := range c.Projects {
		if p.Token == "" {
			return fmt.Errorf("projects[%d].token is required", i)
		}

		if p.ProjectID == "" {
			return fmt.Errorf("projects[%d].project_id is required", i)
		}

		if p.SnapshotName == "" {
			return fmt.Errorf("projects[%d].snapshot_name is required", i)
		}

		if _, dup := seen[p.ProjectID]; dup {
			return fmt.Errorf("projects[%d].project_id %q is duplicated", i, p.ProjectID)
		}

		seen[p.ProjectID] = struct{}{}
	}

	return nil
}

// GetProject returns the project configuration for the given project ID.
// If projectID is empty and there is exactly one project, it returns that project.
func (c *Config) GetProject(projectID string) (*ProjectConfig, error) {
	if projectID == "" {
		if len(c.Projects) == 1 {
			return &c.Projects[0], nil
		}

		return nil, fmt.Errorf("project_id is required when multiple projects are configured")
	}

	for i := range c.Projects {
		if c.Projects[i].ProjectID == projectID {
			return &c.Projects[i], nil
		}
	}

	return nil, fmt.Errorf("project %q not found", projectID)
}
