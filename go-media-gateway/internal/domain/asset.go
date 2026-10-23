package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

const MeterStorageBytes = "storage.bytes"

// Asset is the metadata record for one stored object.
// Bytes live in the blob store, addressed by ContentHash.
type Asset struct {
	ID          string
	TenantID    string
	ContentHash string
	SizeBytes   int64
	MIME        string
	Region      string
	PrimaryNode string
	Replicas    []string
	CreatedAt   time.Time
}

// Usage is the billing event the ledger consumes.
// EventID is stable for a tenant, object, and meter so retries do not double-charge.
type Usage struct {
	EventID    string    `json:"eventId"`
	TenantID   string    `json:"tenantId"`
	AssetID    string    `json:"assetId"`
	Meter      string    `json:"meter"`
	Quantity   int64     `json:"quantity"`
	OccurredAt time.Time `json:"occurredAt"`
}

// AssetID is derived from the tenant and content hash so the same object
// converges on one row across gateway retries.
func AssetID(tenant, hash string) string {
	sum := sha256.Sum256([]byte(tenant + "/" + hash))
	return hex.EncodeToString(sum[:])
}

func StorageEventID(tenant, hash string) string {
	return tenant + ":" + hash + ":" + MeterStorageBytes
}

// ValidTenant matches the billing engine: 1-63 chars, lowercase letters,
// digits, and interior hyphens.
func ValidTenant(id string) bool {
	if len(id) < 1 || len(id) > 63 {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		alnum := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
		if alnum {
			continue
		}
		if c == '-' && i > 0 && i < len(id)-1 {
			continue
		}
		return false
	}
	return true
}

func ValidHash(hash string) bool {
	if len(hash) != 64 {
		return false
	}
	for i := 0; i < len(hash); i++ {
		c := hash[i]
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f':
		default:
			return false
		}
	}
	return true
}

func NormalizeMIME(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if cut, _, ok := strings.Cut(value, ";"); ok {
		value = strings.TrimSpace(cut)
	}
	if !AllowedMIME(value) {
		return "", ErrMIME
	}
	return value, nil
}

func AllowedMIME(value string) bool {
	for _, prefix := range []string{"video/", "audio/", "image/"} {
		if strings.HasPrefix(value, prefix) && len(value) > len(prefix) {
			return true
		}
	}
	return false
}
