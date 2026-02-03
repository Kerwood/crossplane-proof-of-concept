package main

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/google/uuid"
	"k8s.io/utils/ptr"
)

func generateRoleUID(s string) string {
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte(s)).String()
}

func hashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	hash := hex.EncodeToString(h.Sum(nil))
	return hash[:8] // Take first 8 characters
}

// ToSliceOfPtrs converts a slice of values to a slice of pointers.
func ToSliceOfPtrs[T any](slice []T) []*T {
	ptrs := make([]*T, len(slice))
	for i, v := range slice {
		ptrs[i] = ptr.To(v)
	}
	return ptrs
}
