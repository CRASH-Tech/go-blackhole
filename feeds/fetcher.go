package feeds

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

type Fetcher struct {
	URL     string
	Timeout time.Duration
}

func NewFetcher(url string, timeout time.Duration) *Fetcher {
	return &Fetcher{
		URL:     url,
		Timeout: timeout,
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

	return parseIPList(body), nil
}

func parseIPList(data []byte) []string {
	var ips []string
	seen := make(map[string]bool)

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
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
