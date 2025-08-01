// Package types contains types that are common across libnetwork project
package types

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/ishidawataru/sctp"
	"github.com/moby/moby/v2/errdefs"
)

// constants for the IP address type
// Deprecated: use the consts defined in github.com/docker/docker/libnetwork/resolvconf
const (
	IP = iota // IPv4 and IPv6
	IPv4
	IPv6
)

// EncryptionKey is the libnetwork representation of the key distributed by the lead
// manager.
type EncryptionKey struct {
	Subsystem   string
	Algorithm   int32
	Key         []byte
	LamportTime uint64
}

// QosPolicy represents a quality of service policy on an endpoint
type QosPolicy struct {
	MaxEgressBandwidth uint64
}

// TransportPort represents a local Layer 4 endpoint
type TransportPort struct {
	Proto Protocol
	Port  uint16
}

// Equal checks if this instance of TransportPort is equal to the passed one
func (t *TransportPort) Equal(o *TransportPort) bool {
	if t == o {
		return true
	}

	if o == nil {
		return false
	}

	if t.Proto != o.Proto || t.Port != o.Port {
		return false
	}

	return true
}

// GetCopy returns a copy of this TransportPort structure instance
func (t *TransportPort) GetCopy() TransportPort {
	return TransportPort{Proto: t.Proto, Port: t.Port}
}

// String returns the TransportPort structure in string form
func (t *TransportPort) String() string {
	return fmt.Sprintf("%s/%d", t.Proto.String(), t.Port)
}

// PortBinding represents a port binding between the container and the host
type PortBinding struct {
	Proto       Protocol
	IP          net.IP
	Port        uint16
	HostIP      net.IP
	HostPort    uint16
	HostPortEnd uint16
}

// HostAddr returns the host side transport address
func (p PortBinding) HostAddr() (net.Addr, error) {
	switch p.Proto {
	case UDP:
		return &net.UDPAddr{IP: p.HostIP, Port: int(p.HostPort)}, nil
	case TCP:
		return &net.TCPAddr{IP: p.HostIP, Port: int(p.HostPort)}, nil
	case SCTP:
		return &sctp.SCTPAddr{IPAddrs: []net.IPAddr{{IP: p.HostIP}}, Port: int(p.HostPort)}, nil
	default:
		return nil, fmt.Errorf("invalid transport protocol: %s", p.Proto.String())
	}
}

// ContainerAddr returns the container side transport address
func (p PortBinding) ContainerAddr() (net.Addr, error) {
	switch p.Proto {
	case UDP:
		return &net.UDPAddr{IP: p.IP, Port: int(p.Port)}, nil
	case TCP:
		return &net.TCPAddr{IP: p.IP, Port: int(p.Port)}, nil
	case SCTP:
		return &sctp.SCTPAddr{IPAddrs: []net.IPAddr{{IP: p.IP}}, Port: int(p.Port)}, nil
	default:
		return nil, fmt.Errorf("invalid transport protocol: %s", p.Proto.String())
	}
}

// GetCopy returns a copy of this PortBinding structure instance
func (p *PortBinding) GetCopy() PortBinding {
	return PortBinding{
		Proto:       p.Proto,
		IP:          GetIPCopy(p.IP),
		Port:        p.Port,
		HostIP:      GetIPCopy(p.HostIP),
		HostPort:    p.HostPort,
		HostPortEnd: p.HostPortEnd,
	}
}

// Equal returns true if o has the same values as p, else false.
func (p *PortBinding) Equal(o *PortBinding) bool {
	return p.Proto == o.Proto &&
		p.IP.Equal(o.IP) &&
		p.Port == o.Port &&
		p.HostIP.Equal(o.HostIP) &&
		p.HostPort == o.HostPort &&
		p.HostPortEnd == o.HostPortEnd
}

// String returns the PortBinding structure in the form "HostIP:HostPort:IP:Port/Proto",
// omitting un-set fields apart from Port.
func (p PortBinding) String() string {
	var ret strings.Builder
	if len(p.HostIP) > 0 {
		is6 := p.HostIP.To4() == nil
		if is6 {
			ret.WriteRune('[')
		}
		ret.WriteString(p.HostIP.String())
		if is6 {
			ret.WriteRune(']')
		}
		ret.WriteRune(':')
	}
	if p.HostPort != 0 {
		ret.WriteString(strconv.Itoa(int(p.HostPort)))
		if p.HostPortEnd != 0 && p.HostPortEnd != p.HostPort {
			ret.WriteRune('-')
			ret.WriteString(strconv.Itoa(int(p.HostPortEnd)))
		}
	}
	if ret.Len() > 0 {
		ret.WriteRune(':')
	}
	if len(p.IP) > 0 {
		is6 := p.IP.To4() == nil
		if is6 {
			ret.WriteRune('[')
		}
		ret.WriteString(p.IP.String())
		if is6 {
			ret.WriteRune(']')
		}
		ret.WriteRune(':')
	}
	ret.WriteString(strconv.Itoa(int(p.Port)))
	if p.Proto != 0 {
		ret.WriteRune('/')
		ret.WriteString(p.Proto.String())
	}
	return ret.String()
}

const (
	// ICMP is for the ICMP ip protocol
	ICMP = 1
	// TCP is for the TCP ip protocol
	TCP = 6
	// UDP is for the UDP ip protocol
	UDP = 17
	// SCTP is for the SCTP ip protocol
	SCTP = 132
)

// Protocol represents an IP protocol number
type Protocol uint8

func (p Protocol) String() string {
	switch p {
	case ICMP:
		return "icmp"
	case TCP:
		return "tcp"
	case UDP:
		return "udp"
	case SCTP:
		return "sctp"
	default:
		return strconv.Itoa(int(p))
	}
}

// ParseProtocol returns the respective Protocol type for the passed string
func ParseProtocol(s string) Protocol {
	switch strings.ToLower(s) {
	case "icmp":
		return ICMP
	case "udp":
		return UDP
	case "tcp":
		return TCP
	case "sctp":
		return SCTP
	default:
		return 0
	}
}

// GetMacCopy returns a copy of the passed MAC address
func GetMacCopy(from net.HardwareAddr) net.HardwareAddr {
	if from == nil {
		return nil
	}
	to := make(net.HardwareAddr, len(from))
	copy(to, from)
	return to
}

// GetIPCopy returns a copy of the passed IP address
func GetIPCopy(from net.IP) net.IP {
	if from == nil {
		return nil
	}
	to := make(net.IP, len(from))
	copy(to, from)
	return to
}

// GetIPNetCopy returns a copy of the passed IP Network
func GetIPNetCopy(from *net.IPNet) *net.IPNet {
	if from == nil {
		return nil
	}
	bm := make(net.IPMask, len(from.Mask))
	copy(bm, from.Mask)
	return &net.IPNet{IP: GetIPCopy(from.IP), Mask: bm}
}

// GetIPNetCanonical returns the canonical form for the passed network
func GetIPNetCanonical(nw *net.IPNet) *net.IPNet {
	if nw == nil {
		return nil
	}
	c := GetIPNetCopy(nw)
	c.IP = c.IP.Mask(nw.Mask)
	return c
}

// CompareIPNet returns equal if the two IP Networks are equal
func CompareIPNet(a, b *net.IPNet) bool {
	if a == b {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.IP.Equal(b.IP) && bytes.Equal(a.Mask, b.Mask)
}

// IsIPNetValid returns true if the ipnet is a valid network/mask
// combination. Otherwise returns false.
func IsIPNetValid(nw *net.IPNet) bool {
	return nw.String() != "0.0.0.0/0"
}

var v4inV6MaskPrefix = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

// compareIPMask checks if the passed ip and mask are semantically compatible.
// It returns the byte indexes for the address and mask so that caller can
// do bitwise operations without modifying address representation.
func compareIPMask(ip net.IP, mask net.IPMask) (is int, ms int, _ error) {
	// Find the effective starting of address and mask
	if len(ip) == net.IPv6len && ip.To4() != nil {
		is = 12
	}
	if len(ip[is:]) == net.IPv4len && len(mask) == net.IPv6len && bytes.Equal(mask[:12], v4inV6MaskPrefix) {
		ms = 12
	}
	// Check if address and mask are semantically compatible
	if len(ip[is:]) != len(mask[ms:]) {
		return 0, 0, fmt.Errorf("ip and mask are not compatible: (%s, %s)", ip, mask)
	}
	return is, ms, nil
}

// GetHostPartIP returns the host portion of the ip address identified by the mask.
// IP address representation is not modified. If address and mask are not compatible
// an error is returned.
func GetHostPartIP(ip net.IP, mask net.IPMask) (net.IP, error) {
	// Find the effective starting of address and mask
	is, ms, err := compareIPMask(ip, mask)
	if err != nil {
		return nil, fmt.Errorf("cannot compute host portion ip address because %s", err)
	}

	// Compute host portion
	out := GetIPCopy(ip)
	for i := 0; i < len(mask[ms:]); i++ {
		out[is+i] &= ^mask[ms+i]
	}

	return out, nil
}

// GetBroadcastIP returns the broadcast ip address for the passed network (ip and mask).
// IP address representation is not modified. If address and mask are not compatible
// an error is returned.
func GetBroadcastIP(ip net.IP, mask net.IPMask) (net.IP, error) {
	// Find the effective starting of address and mask
	is, ms, err := compareIPMask(ip, mask)
	if err != nil {
		return nil, fmt.Errorf("cannot compute broadcast ip address because %s", err)
	}

	// Compute broadcast address
	out := GetIPCopy(ip)
	for i := 0; i < len(mask[ms:]); i++ {
		out[is+i] |= ^mask[ms+i]
	}

	return out, nil
}

// ParseCIDR returns the *net.IPNet represented by the passed CIDR notation
func ParseCIDR(cidr string) (*net.IPNet, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	ipNet.IP = ip
	return ipNet, nil
}

const (
	// NEXTHOP indicates a StaticRoute with an IP next hop.
	NEXTHOP = iota

	// CONNECTED indicates a StaticRoute with an interface for directly connected peers.
	CONNECTED
)

// StaticRoute is a statically-provisioned IP route.
type StaticRoute struct {
	Destination *net.IPNet

	RouteType int // NEXT_HOP or CONNECTED

	// NextHop will be resolved by the kernel (i.e. as a loose hop).
	NextHop net.IP
}

// GetCopy returns a copy of this StaticRoute structure
func (r *StaticRoute) GetCopy() *StaticRoute {
	d := GetIPNetCopy(r.Destination)
	nh := GetIPCopy(r.NextHop)
	return &StaticRoute{
		Destination: d,
		RouteType:   r.RouteType,
		NextHop:     nh,
	}
}

// InterfaceStatistics represents the interface's statistics
type InterfaceStatistics struct {
	RxBytes   uint64
	RxPackets uint64
	RxErrors  uint64
	RxDropped uint64
	TxBytes   uint64
	TxPackets uint64
	TxErrors  uint64
	TxDropped uint64
}

func (is *InterfaceStatistics) String() string {
	return fmt.Sprintf("\nRxBytes: %d, RxPackets: %d, RxErrors: %d, RxDropped: %d, TxBytes: %d, TxPackets: %d, TxErrors: %d, TxDropped: %d",
		is.RxBytes, is.RxPackets, is.RxErrors, is.RxDropped, is.TxBytes, is.TxPackets, is.TxErrors, is.TxDropped)
}

/******************************
 * Well-known Error Interfaces
 ******************************/

// MaskableError is an interface for errors which can be ignored by caller
type MaskableError interface {
	// Maskable makes implementer into MaskableError type
	Maskable()
}

// InvalidParameterError is an interface for errors originated by a bad request
type InvalidParameterError = errdefs.ErrInvalidParameter

// NotFoundError is an interface for errors raised because a needed resource is not available
type NotFoundError = errdefs.ErrNotFound

// ForbiddenError is an interface for errors which denote a valid request that cannot be honored
type ForbiddenError = errdefs.ErrForbidden

// UnavailableError is an interface for errors returned when the required service is not available
type UnavailableError = errdefs.ErrUnavailable

// NotImplementedError is an interface for errors raised because of requested functionality is not yet implemented
type NotImplementedError = errdefs.ErrNotImplemented

// InternalError is an interface for errors raised because of an internal error
type InternalError interface {
	// Internal makes implementer into InternalError type
	Internal()
}

/******************************
 * Well-known Error Formatters
 ******************************/

// InvalidParameterErrorf creates an instance of InvalidParameterError
func InvalidParameterErrorf(format string, params ...interface{}) error {
	return errdefs.InvalidParameter(fmt.Errorf(format, params...))
}

// NotFoundErrorf creates an instance of NotFoundError
func NotFoundErrorf(format string, params ...interface{}) error {
	return errdefs.NotFound(fmt.Errorf(format, params...))
}

// ForbiddenErrorf creates an instance of ForbiddenError
func ForbiddenErrorf(format string, params ...interface{}) error {
	return errdefs.Forbidden(fmt.Errorf(format, params...))
}

// UnavailableErrorf creates an instance of UnavailableError
func UnavailableErrorf(format string, params ...interface{}) error {
	return errdefs.Unavailable(fmt.Errorf(format, params...))
}

// NotImplementedErrorf creates an instance of NotImplementedError
func NotImplementedErrorf(format string, params ...interface{}) error {
	return errdefs.NotImplemented(fmt.Errorf(format, params...))
}

// InternalErrorf creates an instance of InternalError
func InternalErrorf(format string, params ...interface{}) error {
	return internal(fmt.Sprintf(format, params...))
}

// InternalMaskableErrorf creates an instance of InternalError and MaskableError
func InternalMaskableErrorf(format string, params ...interface{}) error {
	return maskInternal(fmt.Sprintf(format, params...))
}

/***********************
 * Internal Error Types
 ***********************/

type internal string

func (nt internal) Error() string {
	return string(nt)
}
func (nt internal) Internal() {}

type maskInternal string

func (mnt maskInternal) Error() string {
	return string(mnt)
}
func (mnt maskInternal) Internal() {}
func (mnt maskInternal) Maskable() {}
