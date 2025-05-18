package config

import (
	"io/ioutil"
	"log"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	BGP   BGPConfig    `yaml:"bgp"`
	Feeds []FeedConfig `yaml:"feeds"`
	Web   WebConfig    `yaml:"web"`
}

type BGPConfig struct {
	RouterID  string           `yaml:"router_id"`
	LocalAS   uint32           `yaml:"local_as"`
	Neighbors []NeighborConfig `yaml:"neighbors"`
}

type NeighborConfig struct {
	PeerAddress string `yaml:"peer_address"`
	PeerAS      uint32 `yaml:"peer_as"`
}

type FeedConfig struct {
	URL             string `yaml:"url"`
	Community       string `yaml:"community"`
	RefreshInterval string `yaml:"refresh_interval"`
}

type WebConfig struct {
	Listen string `yaml:"listen"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (f *FeedConfig) GetRefreshDuration() time.Duration {
	d, err := time.ParseDuration(f.RefreshInterval)
	if err != nil {
		log.Printf("Invalid refresh interval %s, defaulting to 60s", f.RefreshInterval)
		return 60 * time.Second
	}
	return d
}
