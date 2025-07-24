package policy

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ImagePolicy represents the allowed and blocked container images
type ImagePolicy struct {
	AllowedImages []string `yaml:"allowed_images"`
	BlockedImages []string `yaml:"blocked_images"`
}

// LoadImagePolicy loads an image policy from a YAML file
func LoadImagePolicy(path string) (*ImagePolicy, error) {
	var policy ImagePolicy
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(file, &policy)
	return &policy, err
}

// imageMatches checks if an image matches a pattern, supporting wildcards
func imageMatches(pattern, image string) bool {
	// Simple wildcard support
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(image, strings.TrimSuffix(pattern, "*"))
	}
	return image == pattern
}

// IsAllowed checks if an image is allowed by the policy
// First checks if it's explicitly blocked, then if it's explicitly allowed
// Returns false by default (deny by default)
func IsAllowed(image string, policy *ImagePolicy) bool {
	// Check if the image is in the blocklist
	for _, blocked := range policy.BlockedImages {
		if imageMatches(blocked, image) {
			return false
		}
	}

	if len(policy.AllowedImages) == 0 {
		return true // If no allowed images, allow all
	}

	// Check if the image is in the allowlist
	for _, allowed := range policy.AllowedImages {
		if imageMatches(allowed, image) {
			return true
		}
	}

	// Default deny
	return false
}
