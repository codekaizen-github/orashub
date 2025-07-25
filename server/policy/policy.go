package policy

import (
	"log"
	"os"
	"strings"

	"github.com/a8m/envsubst"
	"gopkg.in/yaml.v3"
)

// ConfigFile represents the configuration file with registry credentials and repository policies
type ConfigFile struct {
	Registries          []RegistryCredentials `yaml:"registries"`
	AllowedRepositories []string              `yaml:"allowed_repositories"`
	BlockedRepositories []string              `yaml:"blocked_repositories"`
}

// RegistryCredentials represents the credentials for a registry
type RegistryCredentials struct {
	Name     string `yaml:"name"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// ImagePolicy represents the allowed and blocked repositories
// Note: Despite the name "ImagePolicy", this is now focused on repository paths rather than images
type ImagePolicy struct {
	AllowedRepositories []string `yaml:"allowed_repositories"`
	BlockedRepositories []string `yaml:"blocked_repositories"`
}

// LoadConfig loads the configuration file with environment variable substitution
func LoadConfig(path string) (*ConfigFile, error) {
	var config ConfigFile

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Substitute environment variables
	expandedData, err := envsubst.Bytes(data)
	if err != nil {
		return nil, err
	}

	// Unmarshal the YAML
	err = yaml.Unmarshal(expandedData, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// GetImagePolicy extracts the repository policy from the configuration
func (c *ConfigFile) GetImagePolicy() *ImagePolicy {
	return &ImagePolicy{
		AllowedRepositories: c.AllowedRepositories,
		BlockedRepositories: c.BlockedRepositories,
	}
}

// repositoryMatches checks if a repository matches a pattern, supporting wildcards
func repositoryMatches(pattern, repository string) bool {
	// Simple wildcard support
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(repository, strings.TrimSuffix(pattern, "*"))
	}
	return pattern == repository
}

// IsAllowed checks if a repository is allowed by the policy
// First checks if it's explicitly blocked, then if it's explicitly allowed
// Returns false by default (deny by default)
func IsAllowed(repository string, policy *ImagePolicy) bool {
	// Log the repository being checked
	log.Printf("Policy check for repository: %s", repository)

	// Check if the repository is in the blocklist
	for _, blocked := range policy.BlockedRepositories {
		if repositoryMatches(blocked, repository) {
			log.Printf("Repository %s matched blocked pattern %s", repository, blocked)
			return false
		}
	}

	if len(policy.AllowedRepositories) == 0 {
		log.Printf("No allowed repositories configured, allowing %s by default", repository)
		return true // If no allowed repositories, allow all
	}

	// Check if the repository is in the allowlist
	for _, allowed := range policy.AllowedRepositories {
		log.Printf("Checking if %s matches allowed pattern %s", repository, allowed)
		if repositoryMatches(allowed, repository) {
			log.Printf("Repository %s matched allowed pattern %s", repository, allowed)
			return true
		}
	}

	// Default deny
	log.Printf("Repository %s did not match any allowed patterns, denying access", repository)
	return false
}
