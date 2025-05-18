package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/CRASH-Tech/go-blackhole/bgp"
	"github.com/CRASH-Tech/go-blackhole/config"
	"github.com/CRASH-Tech/go-blackhole/feeds"
	"github.com/CRASH-Tech/go-blackhole/web"
)

func main() {
	log.Println("Starting BGP Blackhole Announcer")

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	bgpMgr := bgp.NewManager(cfg)
	if err := bgpMgr.Start(); err != nil {
		log.Fatalf("Failed to start BGP: %v", err)
	}

	var processors []*feeds.Processor
	for _, feedCfg := range cfg.Feeds {
		p := feeds.NewProcessor(&feedCfg, bgpMgr, cfg)
		go p.Start()
		processors = append(processors, p)
	}

	webServer := web.NewServer(bgpMgr)
	go func() {
		if err := webServer.Start(cfg.Web.Listen); err != nil {
			log.Fatalf("Web server error: %v", err)
		}
	}()

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	for _, p := range processors {
		p.Stop()
	}

	log.Println("BGP Blackhole Announcer stopped")
}
