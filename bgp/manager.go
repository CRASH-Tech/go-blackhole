package bgp

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/CRASH-Tech/go-blackhole/config"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/server"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type BGPManager struct {
	server *server.BgpServer
	Config *config.Config
}

func NewManager(cfg *config.Config) *BGPManager {
	return &BGPManager{
		server: server.NewBgpServer(),
		Config: cfg,
		// localAS:  cfg.LocalAS,
		// routerID: cfg.RouterID,
	}
}

func (m *BGPManager) Start() error {
	go m.server.Serve()

	log.Printf("Starting BGP with router ID %s and ASN %d", m.Config.BGP.RouterID, m.Config.BGP.LocalAS)
	if err := m.server.StartBgp(context.Background(), &api.StartBgpRequest{
		Global: &api.Global{
			Asn:        m.Config.BGP.LocalAS,
			RouterId:   m.Config.BGP.RouterID,
			ListenPort: -1,
		},
	}); err != nil {
		return fmt.Errorf("failed to start BGP: %w", err)
	}

	for _, neighbor := range m.Config.BGP.Neighbors {
		log.Printf("Adding neighbor %s (AS %d)", neighbor.PeerAddress, neighbor.PeerAS)
		n := &api.Peer{
			Conf: &api.PeerConf{
				NeighborAddress: neighbor.PeerAddress,
				PeerAsn:         neighbor.PeerAS,
			},
		}
		if err := m.server.AddPeer(context.Background(), &api.AddPeerRequest{Peer: n}); err != nil {
			return fmt.Errorf("failed to add peer %s: %w", neighbor.PeerAddress, err)
		}
	}

	return nil
}

func (m *BGPManager) ListRoutes() ([]*Route, error) {
	var routes []*Route
	totalRoutes := 0 // Счетчик всех маршрутов (включая Withdraw)

	fn := func(d *api.Destination) {
		totalRoutes += len(d.Paths)
		for _, path := range d.Paths {
			if !path.IsWithdraw {
				// routes = append(routes, PathToRoute(path))
				route := PathToRoute(path)
				route.Prefix = fmt.Sprintf("%s/%d", route.Prefix, route.PrefixLen)
				routes = append(routes, route)
			}
		}
	}

	err := m.server.ListPath(context.Background(), &api.ListPathRequest{
		TableType: api.TableType_GLOBAL,
		Family:    &api.Family{Afi: api.Family_AFI_IP, Safi: api.Family_SAFI_UNICAST},
	}, fn)

	log.Printf("BGP Route Stats: Total=%d, Active=%d, Withdrawn=%d",
		totalRoutes, len(routes), totalRoutes-len(routes))

	return routes, err
}

func (m *BGPManager) AnnounceRoute(prefix string, community string) error {
	// Парсим префикс с проверкой ошибок
	ip, ipNet, err := parseIPPrefix(prefix)
	if err != nil {
		return fmt.Errorf("invalid prefix %s: %w", prefix, err)
	}

	// Формируем NLRI
	nlri, err := createNLRI(ip, ipNet)
	if err != nil {
		return fmt.Errorf("failed to create NLRI: %w", err)
	}

	// Получаем next hop
	nextHop := m.Config.BGP.RouterID
	if nextHop == "0.0.0.0" {
		return fmt.Errorf("invalid next hop address")
	}

	// Формируем атрибуты
	attrs, err := createPathAttributes(nextHop, community)
	if err != nil {
		return fmt.Errorf("failed to create path attributes: %w", err)
	}

	// Отправляем маршрут
	_, err = m.server.AddPath(context.Background(), &api.AddPathRequest{
		Path: &api.Path{
			Family: &api.Family{Afi: api.Family_AFI_IP, Safi: api.Family_SAFI_UNICAST},
			Nlri:   nlri,
			Pattrs: attrs,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to add path: %w", err)
	}

	//log.Printf("Announced %s with next-hop %s", ipNet.String(), nextHop)
	return nil
}

func parseIPPrefix(prefix string) (net.IP, *net.IPNet, error) {
	// Пробуем распарсить как CIDR
	ip, ipNet, err := net.ParseCIDR(prefix)
	if err == nil {
		return ip, ipNet, nil
	}

	// Пробуем распарсить как отдельный IP
	ip = net.ParseIP(prefix)
	if ip == nil {
		return nil, nil, fmt.Errorf("not a valid IP address or CIDR")
	}

	// Создаем IPNet с маской /32 для IPv4 или /128 для IPv6
	if ip.To4() != nil {
		return ip, &net.IPNet{IP: ip, Mask: net.CIDRMask(32, 32)}, nil
	}
	return ip, &net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}, nil
}

func createNLRI(ip net.IP, ipNet *net.IPNet) (*anypb.Any, error) {
	prefixLen, _ := ipNet.Mask.Size()

	if ip.To4() != nil {
		return anypb.New(&api.IPAddressPrefix{
			Prefix:    ip.String(),
			PrefixLen: uint32(prefixLen),
		})
	}

	return anypb.New(&api.IPAddressPrefix{
		Prefix:    ip.String(),
		PrefixLen: uint32(prefixLen),
	})
}

func createPathAttributes(nextHop, community string) ([]*anypb.Any, error) {
	attrs := []*anypb.Any{
		mustNewAny(&api.OriginAttribute{Origin: 0}), // IGP
		mustNewAny(&api.NextHopAttribute{
			NextHop: nextHop,
		}),
	}

	if community != "" {
		commAttr, err := createCommunityAttribute(community)
		if err != nil {
			return nil, err
		}
		attrs = append(attrs, commAttr)
	}

	return attrs, nil
}

func createCommunityAttribute(community string) (*anypb.Any, error) {
	commValue := parseCommunity(community)
	if commValue == 0 {
		return nil, fmt.Errorf("invalid community format")
	}

	return anypb.New(&api.CommunitiesAttribute{
		Communities: []uint32{commValue},
	})
}

func mustNewAny(msg proto.Message) *anypb.Any {
	a, err := anypb.New(msg)
	if err != nil {
		panic(fmt.Sprintf("failed to convert to Any: %v", err))
	}
	return a
}
