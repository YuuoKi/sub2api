package service

import (
	"net/netip"
)

// assetSpecialPurposePrefixes mirrors the IANA IPv4 and IPv6 Special-Purpose
// Address Registries. The SSRF boundary intentionally denies every registry
// entry, including entries marked globally reachable, because special-purpose
// semantics are not a stable substitute for an ordinary public origin.
//
// Maintenance: compare this table with both IANA registries whenever their
// "Last Updated" value changes. Last compared: 2026-07-17 (registry update
// 2025-10-09).
var assetSpecialPurposePrefixes = []netip.Prefix{
	// IPv4 Special-Purpose Address Registry.
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.31.196.0/24"),
	netip.MustParsePrefix("192.52.193.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("192.175.48.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	// IPv6 Special-Purpose Address Registry.
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("::ffff:0:0/96"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("100:0:0:1::/64"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("2620:4f:8000::/48"),
	netip.MustParsePrefix("3fff::/20"),
	netip.MustParsePrefix("5f00::/16"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
}

func isPublicAssetAddr(addr netip.Addr) bool {
	if !addr.IsValid() || addr.Zone() != "" || addr.Is4In6() || !addr.IsGlobalUnicast() {
		return false
	}
	for _, prefix := range assetSpecialPurposePrefixes {
		if prefix.Contains(addr) {
			return false
		}
	}
	return true
}
