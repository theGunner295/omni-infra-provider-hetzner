// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package hcloud provides a resilient Hetzner Cloud API client wrapper.
package hcloud

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"go.uber.org/zap"
)

// Client is a resilient wrapper around the Hetzner Cloud API client.
type Client struct {
	inner  *hcloud.Client
	logger *zap.Logger
}

// NewClient creates a new resilient Hetzner Cloud API client for the given API token.
func NewClient(token string, logger *zap.Logger) *Client {
	return &Client{
		inner:  hcloud.NewClient(hcloud.WithToken(token)),
		logger: logger,
	}
}

// NewClientFromInner creates a Client wrapping an existing hcloud.Client (useful for testing).
func NewClientFromInner(inner *hcloud.Client, logger *zap.Logger) *Client {
	return &Client{
		inner:  inner,
		logger: logger,
	}
}

// Inner returns the underlying hcloud.Client for direct access when needed.
func (c *Client) Inner() *hcloud.Client {
	return c.inner
}

// Do executes an API operation with exponential backoff retry logic.
func (c *Client) Do(ctx context.Context, operation string, fn func() error) error {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 500 * time.Millisecond
	b.MaxInterval = 30 * time.Second
	b.MaxElapsedTime = 5 * time.Minute

	bCtx := backoff.WithContext(b, ctx)

	attempt := 0

	return backoff.Retry(func() error {
		attempt++

		err := fn()
		if err == nil {
			return nil
		}

		// For hcloud API errors, retry only on known transient codes; treat all others as permanent.
		var hcloudErr hcloud.Error
		if errors.As(err, &hcloudErr) {
			switch hcloudErr.Code {
			case hcloud.ErrorCodeRateLimitExceeded,
				hcloud.ErrorCodeServiceError,
				hcloud.ErrorCodeConflict:
				c.logger.Warn("hetzner API transient error, retrying",
					zap.String("operation", operation),
					zap.String("code", string(hcloudErr.Code)),
					zap.Int("attempt", attempt),
					zap.Error(err),
				)

				return err // retryable
			}

			// All other API errors are non-retryable.
			return backoff.Permanent(err)
		}

		// Retry on network/connection errors (not hcloud.Error).
		c.logger.Warn("hetzner API request failed, retrying",
			zap.String("operation", operation),
			zap.Int("attempt", attempt),
			zap.Error(err),
		)

		return err
	}, bCtx)
}

// FindSnapshotByName finds a snapshot image by name in the Hetzner Cloud account.
// Returns the image or an error if not found.
func (c *Client) FindSnapshotByName(ctx context.Context, name string) (*hcloud.Image, error) {
	var result *hcloud.Image

	err := c.Do(ctx, "FindSnapshotByName", func() error {
		opts := hcloud.ImageListOpts{
			Type: []hcloud.ImageType{hcloud.ImageTypeSnapshot},
			ListOpts: hcloud.ListOpts{
				LabelSelector: "",
			},
		}

		images, _, err := c.inner.Image.List(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to list images: %w", err)
		}

		for _, img := range images {
			if img.Name == name || img.Description == name {
				result = img

				return nil
			}
		}

		return backoff.Permanent(fmt.Errorf("snapshot %q not found", name))
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// ValidateSKUAndLocation validates that the given server type is available in the given location.
func (c *Client) ValidateSKUAndLocation(ctx context.Context, serverType, location string) error {
	return c.Do(ctx, "ValidateSKUAndLocation", func() error {
		st, _, err := c.inner.ServerType.GetByName(ctx, serverType)
		if err != nil {
			return fmt.Errorf("failed to get server type %q: %w", serverType, err)
		}

		if st == nil {
			return backoff.Permanent(fmt.Errorf("server type %q not found", serverType))
		}

		loc, _, err := c.inner.Location.GetByName(ctx, location)
		if err != nil {
			return fmt.Errorf("failed to get location %q: %w", location, err)
		}

		if loc == nil {
			return backoff.Permanent(fmt.Errorf("location %q not found", location))
		}

		// Check if server type is available in the location
		for _, stLoc := range st.Locations {
			if stLoc.Location != nil && stLoc.Location.Name == location {
				return nil
			}
		}

		return backoff.Permanent(fmt.Errorf("server type %q is not available in location %q", serverType, location))
	})
}

// FindNetworkByName finds a private network by name.
func (c *Client) FindNetworkByName(ctx context.Context, name string) (*hcloud.Network, error) {
	var result *hcloud.Network

	err := c.Do(ctx, "FindNetworkByName", func() error {
		network, _, err := c.inner.Network.GetByName(ctx, name)
		if err != nil {
			return fmt.Errorf("failed to get network %q: %w", name, err)
		}

		if network == nil {
			return backoff.Permanent(fmt.Errorf("network %q not found", name))
		}

		result = network

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetServerByID returns a server by its integer ID string.
func (c *Client) GetServerByID(ctx context.Context, serverID int64) (*hcloud.Server, error) {
	var result *hcloud.Server

	err := c.Do(ctx, "GetServerByID", func() error {
		server, _, err := c.inner.Server.GetByID(ctx, serverID)
		if err != nil {
			return fmt.Errorf("failed to get server %d: %w", serverID, err)
		}

		result = server

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// IsNotFound returns true if the error represents a 404 not found response.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	var hcloudErr hcloud.Error
	if errors.As(err, &hcloudErr) {
		return hcloudErr.Code == hcloud.ErrorCodeNotFound
	}

	// hcloud-go also returns nil response with nil error for not-found in some Get methods
	return false
}

// IsHTTPNotFound checks if the response status is 404.
func IsHTTPNotFound(resp *hcloud.Response) bool {
	return resp != nil && resp.Response != nil && resp.Response.StatusCode == http.StatusNotFound
}
