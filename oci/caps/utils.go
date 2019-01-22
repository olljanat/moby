package caps // import "github.com/docker/docker/oci/caps"

import (
	"fmt"
	"strings"

	"github.com/docker/docker/errdefs"
	"github.com/syndtr/gocapability/capability"
)

var capabilityList Capabilities

func init() {
	last := capability.CAP_LAST_CAP
	// hack for RHEL6 which has no /proc/sys/kernel/cap_last_cap
	if last == capability.Cap(63) {
		last = capability.CAP_BLOCK_SUSPEND
	}
	for _, cap := range capability.List() {
		if cap > last {
			continue
		}
		capabilityList = append(capabilityList,
			&CapabilityMapping{
				Key:   "CAP_" + strings.ToUpper(cap.String()),
				Value: cap,
			},
		)
	}
}

type (
	// CapabilityMapping maps linux capability name to its value of capability.Cap type
	// Capabilities is one of the security systems in Linux Security Module (LSM)
	// framework provided by the kernel.
	// For more details on capabilities, see http://man7.org/linux/man-pages/man7/capabilities.7.html
	CapabilityMapping struct {
		Key   string         `json:"key,omitempty"`
		Value capability.Cap `json:"value,omitempty"`
	}
	// Capabilities contains all CapabilityMapping
	Capabilities []*CapabilityMapping
)

// String returns <key> of CapabilityMapping
func (c *CapabilityMapping) String() string {
	return c.Key
}

// GetCapability returns CapabilityMapping which contains specific key
func GetCapability(key string) *CapabilityMapping {
	for _, capp := range capabilityList {
		if capp.Key == key {
			cpy := *capp
			return &cpy
		}
	}
	return nil
}

// GetAllCapabilities returns all of the capabilities
func GetAllCapabilities() []string {
	output := make([]string, len(capabilityList))
	for i, capability := range capabilityList {
		output[i] = capability.String()
	}
	return output
}

// inSlice tests whether a string is contained in a slice of strings or not.
// Case sensitive comparisation is used for Capabilities field
// Case insensitive comparisation is used for CapAdd and CapDrop
func inSlice(slice []string, s string) bool {
	for _, ss := range slice {
		if s == ss {
			return true
		}
	}
	return false
}

const allCapabilities = "ALL"

// NormalizeLegacyCapabilities normalizes, and validates CapAdd/CapDrop capabilities
// by upper-casing them, and adding a CAP_ prefix (if not yet present).
//
// This function also accepts the "ALL" magic-value, that's used by CapAdd/CapDrop.
func NormalizeLegacyCapabilities(caps []string) ([]string, error) {
	var normalized []string

	valids := GetAllCapabilities()
	for _, c := range caps {
		c = strings.ToUpper(c)
		if c == allCapabilities {
			normalized = append(normalized, c)
			continue
		}
		if !strings.HasPrefix(c, "CAP_") {
			c = "CAP_" + c
		}
		if !inSlice(valids, c) {
			return nil, errdefs.InvalidParameter(fmt.Errorf("unknown capability: %q", c))
		}
	}
	return normalized, nil
}

// ValidateCapabilities validates if caps only contains valid capabilities
func ValidateCapabilities(caps []string) error {
	valids := GetAllCapabilities()
	for _, c := range caps {
		if !inSlice(valids, c) {
			return errdefs.InvalidParameter(fmt.Errorf("unknown capability: %q", c))
		}
	}
	return nil
}

// TweakCapabilities can tweak capabilities by adding or dropping capabilities
// based on the basics capabilities.
func TweakCapabilities(basics, adds, drops []string, capabilities []string, privileged bool) ([]string, error) {
	var (
		newCaps []string
		allCaps = GetAllCapabilities()
	)

	if privileged {
		return allCaps, nil
	}

	if capabilities != nil {
		if err := ValidateCapabilities(capabilities); err != nil {
			return nil, err
		}
		return capabilities, nil
	}

	capDrop, err := NormalizeLegacyCapabilities(drops)
	if err != nil {
		return nil, err
	}
	capAdd, err := NormalizeLegacyCapabilities(adds)
	if err != nil {
		return nil, err
	}

	// handle --cap-add=all
	if inSlice(capAdd, allCapabilities) {
		basics = allCaps
	}

	if !inSlice(capDrop, allCapabilities) {
		for _, c := range basics {
			// skip `all` already handled above
			if c == allCapabilities {
				continue
			}
			// if we don't drop `all`, add back all the non-dropped caps
			if !inSlice(capDrop, c) {
				newCaps = append(newCaps, c)
			}
		}
	}

	for _, c := range adds {
		// skip `all` already handled above
		if c == allCapabilities {
			continue
		}
		// add cap if not already in the list
		if !inSlice(newCaps, c) {
			newCaps = append(newCaps, c)
		}
	}
	return newCaps, nil
}
