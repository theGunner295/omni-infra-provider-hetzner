// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hcloud_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	hcloudlib "github.com/hetznercloud/hcloud-go/v2/hcloud"
	"go.uber.org/zap"

	hclient "github.com/theGunner295/omni-infra-provider-hetzner/internal/pkg/hcloud"
)

// newTestClient creates a hcloud.Client pointing at the provided test server.
func newTestClient(t *testing.T, server *httptest.Server) *hclient.Client {
	t.Helper()

	inner := hcloudlib.NewClient(
		hcloudlib.WithToken("test-token"),
		hcloudlib.WithEndpoint(server.URL),
	)

	return hclient.NewClientFromInner(inner, zap.NewNop())
}

// jsonOK writes a JSON response with HTTP 200.
func jsonOK(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(body) //nolint:errcheck
}

func TestClientDoRetryOnTransientErrors(t *testing.T) {
	var attempts atomic.Int32

	// Simulate a function that fails twice then succeeds
	err := func() error {
		client := hclient.NewClientFromInner(
			hcloudlib.NewClient(hcloudlib.WithToken("tok")),
			zap.NewNop(),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		return client.Do(ctx, "test", func() error {
			n := attempts.Add(1)
			if n < 3 {
				// Return a transient error (not permanent)
				return fmt.Errorf("transient network error")
			}

			return nil
		})
	}()

	if err != nil {
		t.Fatalf("client.Do returned error after retries: %v", err)
	}

	if attempts.Load() < 3 {
		t.Fatalf("expected at least 3 attempts for transient error, got %d", attempts.Load())
	}
}

func TestClientDoNoRetryOnPermanentError(t *testing.T) {
	var callCount int

	client := hclient.NewClientFromInner(
		hcloudlib.NewClient(hcloudlib.WithToken("tok")),
		zap.NewNop(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Do(ctx, "testNotFound", func() error {
		callCount++

		// Permanent error: wrap with backoff.Permanent via hcloud.Error not_found
		return hcloudlib.Error{
			Code:    hcloudlib.ErrorCodeNotFound,
			Message: "server not found",
		}
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// With a non-retryable API error (not_found/invalid_input/etc), it should only be called once
	if callCount > 1 {
		t.Fatalf("expected 1 call for permanent error, got %d", callCount)
	}
}

func TestClientDoRetryOnConflict(t *testing.T) {
	var callCount int

	client := hclient.NewClientFromInner(
		hcloudlib.NewClient(hcloudlib.WithToken("tok")),
		zap.NewNop(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := client.Do(ctx, "testConflict", func() error {
		callCount++
		if callCount < 3 {
			return hcloudlib.Error{
				Code:    hcloudlib.ErrorCodeConflict,
				Message: "conflict: resource changed during request",
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("expected eventual success after conflict retries, got: %v", err)
	}

	if callCount < 3 {
		t.Fatalf("expected at least 3 calls for conflict retry, got %d", callCount)
	}
}

func TestClientFindSnapshotByName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images" {
			http.NotFound(w, r)

			return
		}

		jsonOK(w, map[string]any{
			"images": []map[string]any{
				{"id": 1, "name": "talos-v1.9", "description": "Talos v1.9", "type": "snapshot", "status": "available"},
				{"id": 2, "name": "other-image", "description": "Some other image", "type": "snapshot", "status": "available"},
			},
			"meta": map[string]any{
				"pagination": map[string]any{
					"page":          1,
					"per_page":      25,
					"total_entries": 2,
					"total_pages":   1,
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	img, err := client.FindSnapshotByName(ctx, "talos-v1.9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if img == nil {
		t.Fatal("expected image, got nil")
	}

	if img.Name != "talos-v1.9" {
		t.Fatalf("expected name talos-v1.9, got %q", img.Name)
	}
}

func TestClientFindSnapshotByNameNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images" {
			http.NotFound(w, r)

			return
		}

		jsonOK(w, map[string]any{
			"images": []map[string]any{},
			"meta": map[string]any{
				"pagination": map[string]any{
					"page":          1,
					"per_page":      25,
					"total_entries": 0,
					"total_pages":   1,
				},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := client.FindSnapshotByName(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for not found snapshot, got nil")
	}
}

