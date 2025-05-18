package feeds

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/CRASH-Tech/go-blackhole/config"
)

type Fetcher struct {
	URL     string
	Timeout time.Duration
	Config  config.Config
}

func NewFetcher(url string, timeout time.Duration, cfg *config.Config) *Fetcher {
	return &Fetcher{
		URL:     url,
		Timeout: timeout,
		Config:  *cfg,
	}
}

func (f *Fetcher) Fetch() ([]string, error) {
	client := &http.Client{Timeout: f.Timeout}
	resp, err := client.Get(f.URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseIPList(body, &f.Config), nil
}

func parseIPList(data []byte, cfg *config.Config) []string {
	var ips []string
	seen := make(map[string]bool)

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if isPrefixWhiteListed(line, cfg) {
			log.Printf("Prefix %s is whitelisted. Ignore it.", line)
			continue
		}

		// Нормализуем формат (добавляем /32 для IPv4 и /128 для IPv6 если нужно)
		normalized := normalizeIPPrefix(line)
		if normalized == "" {
			continue
		}

		// Удаляем дубликаты
		if !seen[normalized] {
			ips = append(ips, normalized)
			seen[normalized] = true
		} else {
			log.Printf("Found duplicated prefix: %s", normalized)
		}
	}
	return ips
}

func normalizeIPPrefix(prefix string) string {
	// Пробуем распарсить как CIDR
	if _, _, err := net.ParseCIDR(prefix); err == nil {
		return prefix
	}

	// Пробуем распарсить как IP
	ip := net.ParseIP(prefix)
	if ip == nil {
		return ""
	}

	// Добавляем маску
	if ip.To4() != nil {
		return prefix + "/32"
	}
	return prefix + "/128"
}

func maskContains(outer, inner net.IPMask) bool {
	outerOnes, _ := outer.Size()
	innerOnes, _ := inner.Size()
	return outerOnes <= innerOnes
}

func isPrefixWhiteListed(prefix string, cfg *config.Config) bool {
	ip := net.ParseIP(prefix)
	_, inputNet, errNet := net.ParseCIDR(prefix)

	if ip == nil && errNet != nil {
		log.Printf("Something wrong with %s", prefix)
		return true // Некорректный формат
	}

	for _, subnetStr := range cfg.Whitelist {
		if prefix == subnetStr {
			return true
		}
		_, subnet, err := net.ParseCIDR(subnetStr)
		if err != nil {
			log.Printf("Cannot parce CIDR: %s", subnetStr)
			return true
		}

		if ip != nil {
			if subnet.Contains(ip) {
				return true
			}
		} else if inputNet != nil {
			if subnet.Contains(inputNet.IP) && maskContains(subnet.Mask, inputNet.Mask) {
				return true
			}
		}
	}

	return false
}
