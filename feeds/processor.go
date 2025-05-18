package feeds

import (
	"log"
	"sync"
	"time"

	"github.com/CRASH-Tech/go-blackhole/bgp"
	"github.com/CRASH-Tech/go-blackhole/config"
)

type GlobalStats struct {
	mu           sync.Mutex
	ActiveRoutes int            // Только активные маршруты
	FeedStats    map[string]int // Статистика по каждому feed
}

var (
	globalStats = GlobalStats{
		FeedStats: make(map[string]int),
	}
)

type Processor struct {
	fetcher   *Fetcher
	bgpMgr    *bgp.BGPManager
	community string
	interval  time.Duration
	stopChan  chan struct{}
}

func NewProcessor(feed *config.FeedConfig, bgpMgr *bgp.BGPManager, cfg *config.Config) *Processor {
	return &Processor{
		fetcher:   NewFetcher(feed.URL, 10*time.Second, cfg),
		bgpMgr:    bgpMgr,
		community: feed.Community,
		interval:  feed.GetRefreshDuration(),
		stopChan:  make(chan struct{}),
	}
}

func (p *Processor) Start() {
	log.Printf("Starting feed processor for %s with interval %v", p.fetcher.URL, p.interval)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.process()
		case <-p.stopChan:
			log.Println("Stopping feed processor")
			return
		}
	}
}

func (p *Processor) Stop() {
	close(p.stopChan)
}

func (p *Processor) process() {
	ips, err := p.fetcher.Fetch()
	if err != nil {
		log.Printf("[Feed %s] Error fetching IP list: %v", p.fetcher.URL, err)
		return
	}

	// Получаем текущие маршруты для этого feed
	currentCount := p.getCurrentFeedCount()

	successCount := 0
	for _, ip := range ips {
		if err := p.bgpMgr.AnnounceRoute(ip, p.community); err != nil {
			log.Printf("[Feed %s] Failed to announce %s: %v", p.fetcher.URL, ip, err)
		} else {
			successCount++
		}
	}

	// Обновляем статистику
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()

	// Корректируем общее количество
	globalStats.ActiveRoutes += successCount - currentCount
	// Обновляем статистику по конкретному feed
	globalStats.FeedStats[p.fetcher.URL] = successCount

	log.Printf("[Feed %s] Results: New=%d, Previous=%d, Active=%d",
		p.fetcher.URL, successCount, currentCount, globalStats.FeedStats[p.fetcher.URL])
	log.Printf("[Global] Total active routes: %d", globalStats.ActiveRoutes)
}

func (p *Processor) getCurrentFeedCount() int {
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()
	return globalStats.FeedStats[p.fetcher.URL]
}

func GetActiveRoutes() int {
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()
	return globalStats.ActiveRoutes
}

func GetFeedStats() map[string]int {
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()

	// Возвращаем копию, чтобы избежать гонок данных
	stats := make(map[string]int)
	for k, v := range globalStats.FeedStats {
		stats[k] = v
	}
	return stats
}
