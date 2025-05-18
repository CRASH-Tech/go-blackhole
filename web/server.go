package web

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/CRASH-Tech/go-blackhole/bgp"
)

type Server struct {
	bgpMgr *bgp.BGPManager
}

func NewServer(bgpMgr *bgp.BGPManager) *Server {
	return &Server{bgpMgr: bgpMgr}
}

func (s *Server) Start(listenAddr string) error {
	log.Printf("Starting web server on %s", listenAddr)
	http.HandleFunc("/inbound", s.handleInbound)
	http.HandleFunc("/outbound", s.handleOutbound)

	return http.ListenAndServe(listenAddr, nil)
}

func (s *Server) handleInbound(w http.ResponseWriter, r *http.Request) {
	routes, err := s.bgpMgr.ListRoutes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var result []string
	for _, route := range routes {
		if route.NextHop != s.bgpMgr.Config.BGP.RouterID {
			result = append(result, route.Prefix)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleOutbound(w http.ResponseWriter, r *http.Request) {
	routes, err := s.bgpMgr.ListRoutes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var result []string
	for _, route := range routes {
		if route.NextHop == s.bgpMgr.Config.BGP.RouterID {
			result = append(result, route.Prefix)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
