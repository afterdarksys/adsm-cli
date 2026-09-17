package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type envelope[T any] struct {
	Data T `json:"data"`
}
type Membership struct {
	OrganizationID string `json:"organization_id"`
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	Type           string `json:"type"`
}
type Service struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	State         string `json:"state"`
	Available     bool   `json:"available"`
	Authorization string `json:"authorization"`
	ControlPlane  string `json:"control_plane,omitempty"`
	DataPlane     string `json:"data_plane,omitempty"`
}
type Resource struct {
	ID             string         `json:"id"`
	Service        string         `json:"service"`
	ResourceType   string         `json:"resource_type"`
	CustomerName   string         `json:"customer_name"`
	LifecycleState string         `json:"lifecycle_state"`
	Revision       int64          `json:"revision"`
	DesiredState   map[string]any `json:"desired_state"`
	ObservedState  map[string]any `json:"observed_state"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}
type Operation struct {
	ID           string         `json:"id"`
	OperationKey string         `json:"operation_key"`
	ResourceID   string         `json:"resource_id"`
	State        string         `json:"state"`
	Result       map[string]any `json:"result"`
	Error        map[string]any `json:"error"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}
type StorageCredential struct {
	AccessKey    string           `json:"access_key_id"`
	SecretKey    string           `json:"secret_access_key,omitempty"`
	SessionToken string           `json:"session_token,omitempty"`
	Expiration   time.Time        `json:"expiration"`
	PrincipalID  string           `json:"principal_id,omitempty"`
	Buckets      []map[string]any `json:"buckets,omitempty"`
}

func (c *Client) Context(ctx context.Context) (map[string]any, error) {
	var response envelope[map[string]any]
	err := c.Do(ctx, http.MethodGet, "/v1/adsm/context", nil, &response)
	return response.Data, err
}

func (c *Client) Memberships(ctx context.Context) ([]Membership, error) {
	var response envelope[struct {
		Memberships []Membership `json:"memberships"`
	}]
	err := c.Do(ctx, http.MethodGet, "/v1/platform/memberships", nil, &response)
	return response.Data.Memberships, err
}
func (c *Client) Services(ctx context.Context) ([]Service, string, error) {
	var response envelope[struct {
		Services            []Service `json:"services"`
		EntitlementRevision string    `json:"entitlement_revision"`
	}]
	err := c.Do(ctx, http.MethodGet, "/v1/adsm/services", nil, &response)
	return response.Data.Services, response.Data.EntitlementRevision, err
}
func (c *Client) Buckets(ctx context.Context) ([]Resource, error) {
	var response envelope[struct {
		Buckets []Resource `json:"buckets"`
	}]
	err := c.Do(ctx, http.MethodGet, "/v1/adsm/storage/buckets", nil, &response)
	return response.Data.Buckets, err
}
func (c *Client) Bucket(ctx context.Context, name string) (Resource, error) {
	var response envelope[struct {
		Bucket Resource `json:"bucket"`
	}]
	err := c.Do(ctx, http.MethodGet, "/v1/adsm/storage/buckets/"+url.PathEscape(name), nil, &response)
	return response.Data.Bucket, err
}
func (c *Client) CreateBucket(ctx context.Context, name, region, key string) (Operation, error) {
	var response envelope[struct {
		Operation Operation `json:"operation"`
	}]
	headers := http.Header{"Idempotency-Key": {key}}
	err := c.DoHeaders(ctx, http.MethodPost, "/v1/adsm/storage/buckets", map[string]any{"name": name, "region": region}, &response, headers)
	return response.Data.Operation, err
}
func (c *Client) DeleteBucket(ctx context.Context, name, key string, revision int64) (Operation, error) {
	var response envelope[struct {
		Operation Operation `json:"operation"`
	}]
	headers := http.Header{"Idempotency-Key": {key}, "If-Match": {strconv.FormatInt(revision, 10)}}
	err := c.DoHeaders(ctx, http.MethodDelete, "/v1/adsm/storage/buckets/"+url.PathEscape(name), nil, &response, headers)
	return response.Data.Operation, err
}
func (c *Client) Operation(ctx context.Context, id string) (Operation, error) {
	var response envelope[struct {
		Operation Operation `json:"operation"`
	}]
	err := c.Do(ctx, http.MethodGet, "/v1/adsm/operations/"+url.PathEscape(id), nil, &response)
	return response.Data.Operation, err
}
func (c *Client) WaitOperation(ctx context.Context, id string, interval time.Duration) (Operation, error) {
	if interval <= 0 {
		interval = time.Second
	}
	for {
		operation, err := c.Operation(ctx, id)
		if err != nil {
			return Operation{}, err
		}
		switch operation.State {
		case "succeeded":
			return operation, nil
		case "failed", "cancelled":
			return operation, fmt.Errorf("operation %s %s: %v", operation.ID, operation.State, operation.Error)
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return Operation{}, ctx.Err()
		case <-timer.C:
		}
	}
}
func (c *Client) IssueStorageCredential(ctx context.Context, buckets, actions []string, duration int64, key string) (StorageCredential, error) {
	var response envelope[struct {
		Credential StorageCredential `json:"credential"`
	}]
	headers := http.Header{"Idempotency-Key": {key}}
	err := c.DoHeaders(ctx, http.MethodPost, "/v1/adsm/storage/credentials", map[string]any{"buckets": buckets, "actions": actions, "duration_seconds": duration}, &response, headers)
	return response.Data.Credential, err
}
func (c *Client) StorageCredentials(ctx context.Context) ([]StorageCredential, error) {
	var response envelope[struct {
		Credentials []StorageCredential `json:"credentials"`
	}]
	err := c.Do(ctx, http.MethodGet, "/v1/adsm/storage/credentials", nil, &response)
	return response.Data.Credentials, err
}
func (c *Client) RevokeStorageCredential(ctx context.Context, accessKey, key string) error {
	headers := http.Header{"Idempotency-Key": {key}}
	return c.DoHeaders(ctx, http.MethodDelete, "/v1/adsm/storage/credentials/"+url.PathEscape(accessKey), nil, nil, headers)
}
