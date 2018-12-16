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
func inSlice(slice []string, s string, caseSensitive bool) bool {
	for _, ss := range slice {
		if !caseSensitive {
			s = strings.ToUpper(s)
			ss = strings.ToUpper(ss)
		}
		if s == ss {
			return true
		}
	}
	return false
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

	if (len(adds) > 0 || len(drops) > 0) && capabilities != nil {
		return nil, errdefs.InvalidParameter(fmt.Errorf("conflicting options: Capabilities and CapAdd / CapDrop"))
	}

	if capabilities != nil {
		for _, cap := range capabilities {
			if !inSlice(allCaps, cap, true) {
				return nil, errdefs.InvalidParameter(fmt.Errorf("Unknown capability: %q", cap))
			}
			newCaps = append(newCaps, cap)
		}
		return newCaps, nil
	}

	// look for invalid cap in the drop list
	for _, cap := range drops {
		if strings.ToUpper(cap) == "ALL" {
			continue
		}
		if !strings.HasPrefix(cap, "CAP_") {
			cap = "CAP_" + cap
		}
		if !inSlice(allCaps, cap, false) {
			return nil, errdefs.InvalidParameter(fmt.Errorf("Unknown capability to drop: %q", cap))
		}
	}

	// handle --cap-add=all
	if inSlice(adds, "ALL", false) {
		basics = allCaps
	}

	if !inSlice(drops, "ALL", false) {
		for _, cap := range basics {
			// skip `all` already handled above
			if strings.ToUpper(cap) == "ALL" {
				continue
			}
			// if we don't drop `all`, add back all the non-dropped caps
			if !inSlice(drops, cap[4:], false) {
				if !strings.HasPrefix(cap, "CAP_") {
					cap = "CAP_" + cap
				}
				newCaps = append(newCaps, strings.ToUpper(cap))
			}
		}
	}

	for _, cap := range adds {
		// skip `all` already handled above
		if strings.ToUpper(cap) == "ALL" {
			continue
		}
		if !strings.HasPrefix(cap, "CAP_") {
			cap = "CAP_" + cap
		}
		if !inSlice(allCaps, cap, false) {
			return nil, errdefs.InvalidParameter(fmt.Errorf("Unknown capability to add: %q", cap))
		}
		// add cap if not already in the list
		if !inSlice(newCaps, cap, false) {
			newCaps = append(newCaps, strings.ToUpper(cap))
		}
	}
	return newCaps, nil
}
