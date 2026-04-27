//go:build playwright

package browser

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/playwright-community/playwright-go"
)

func applyAdblockRules(context playwright.BrowserContext, rulesPath string) error {
	rulesPath = strings.TrimSpace(rulesPath)
	if rulesPath == "" {
		return nil
	}
	data, err := os.Open(rulesPath)
	if err != nil {
		return fmt.Errorf("open adblock rules %q: %w", rulesPath, err)
	}
	defer data.Close()

	rules, err := parseAdblockDomainRules(data)
	if err != nil {
		return err
	}
	if len(rules) == 0 {
		return nil
	}

	return context.Route("**/*", func(route playwright.Route) {
		request := route.Request()
		if request == nil {
			_ = route.Continue()
			return
		}
		if shouldBlockAdblockRequest(request.URL(), rules) {
			_ = route.Abort()
			return
		}
		_ = route.Continue()
	})
}

func parseAdblockDomainRules(file *os.File) ([]string, error) {
	scanner := bufio.NewScanner(file)
	rules := make([]string, 0, 256)
	seen := make(map[string]struct{})
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "[") {
			continue
		}
		if strings.HasPrefix(line, "@@") {
			continue
		}
		domain := strings.TrimPrefix(line, "||")
		domain = strings.TrimPrefix(domain, "|")
		domain = strings.TrimSuffix(domain, "^")
		domain = strings.TrimPrefix(domain, "https://")
		domain = strings.TrimPrefix(domain, "http://")
		domain = strings.TrimPrefix(domain, "www.")
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}
		if idx := strings.IndexAny(domain, "/?#"); idx >= 0 {
			domain = domain[:idx]
		}
		domain = strings.ToLower(domain)
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		rules = append(rules, domain)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return rules, nil
}

func shouldBlockAdblockRequest(rawURL string, rules []string) bool {
	if rawURL == "" {
		return false
	}
	lowerURL := strings.ToLower(rawURL)
	for _, rule := range rules {
		if rule == "" {
			continue
		}
		if strings.Contains(lowerURL, rule) {
			return true
		}
	}
	return false
}
