package bgp

import (
	"fmt"
	"strings"

	api "github.com/osrg/gobgp/v3/api"
)

type Route struct {
	Prefix      string   `json:"prefix"`
	PrefixLen   uint32   `json:"prefix_len"`
	NextHop     string   `json:"next_hop"`
	AsPath      []uint32 `json:"as_path"`
	Communities []uint32 `json:"communities"`
}

func PathToRoute(path *api.Path) *Route {
	r := &Route{
		AsPath:      make([]uint32, 0),
		Communities: make([]uint32, 0),
	}

	// Парсим NLRI (префикс сети)
	if nlri := path.GetNlri(); nlri != nil {
		prefix := &api.IPAddressPrefix{}
		if err := nlri.UnmarshalTo(prefix); err == nil {
			r.Prefix = prefix.Prefix
			r.PrefixLen = prefix.PrefixLen
		}
	}

	// Парсим атрибуты BGP
	for _, attr := range path.Pattrs {
		switch attr.TypeUrl {
		case "type.googleapis.com/apipb.NextHopAttribute":
			nh := &api.NextHopAttribute{}
			if err := attr.UnmarshalTo(nh); err == nil {
				r.NextHop = nh.NextHop
			}

		case "type.googleapis.com/apipb.AsPathAttribute":
			asPath := &api.AsPathAttribute{}
			if err := attr.UnmarshalTo(asPath); err == nil && len(asPath.Segments) > 0 {
				r.AsPath = asPath.Segments[0].Numbers
			}

		case "type.googleapis.com/apipb.CommunitiesAttribute":
			comm := &api.CommunitiesAttribute{}
			if err := attr.UnmarshalTo(comm); err == nil {
				r.Communities = comm.Communities
			}
		}
	}

	return r
}

func parseCommunity(community string) uint32 {
	parts := strings.Split(community, ":")
	if len(parts) != 2 {
		return 0
	}

	var asn, val uint32
	fmt.Sscanf(parts[0], "%d", &asn)
	fmt.Sscanf(parts[1], "%d", &val)
	return (asn << 16) | val
}
