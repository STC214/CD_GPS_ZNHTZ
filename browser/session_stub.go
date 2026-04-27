//go:build !playwright

package browser

import (
	"errors"
	"os"
	"strings"
)

func openFirefoxSession(m FirefoxMiddleware, opts BrowserSessionOptions) (*FirefoxSession, error) {
	spec := m.LaunchSpec(opts)
	if strings.TrimSpace(spec.URL) == "" {
		return nil, errors.New("browser middleware url is empty")
	}
	if strings.TrimSpace(spec.BrowserPath) == "" {
		return nil, errors.New("browser path is empty")
	}
	if _, err := os.Stat(spec.BrowserPath); err != nil {
		return nil, err
	}
	if _, err := os.Stat(spec.StealthScript.Path); err != nil {
		return nil, err
	}
	return nil, errors.New("playwright runtime is disabled in this build; use -tags playwright")
}

func closeFirefoxSession(s *FirefoxSession) error {
	return nil
}

func sessionTitle(s *FirefoxSession) (string, error) {
	return "", errors.New("playwright runtime is disabled in this build; use -tags playwright")
}

func waitFirefoxSessionClosed(s *FirefoxSession) error {
	return errors.New("playwright runtime is disabled in this build; use -tags playwright")
}

func sessionContent(s *FirefoxSession) (string, error) {
	return "", errors.New("playwright runtime is disabled in this build; use -tags playwright")
}

func sessionGoto(s *FirefoxSession, url string) error {
	return errors.New("playwright runtime is disabled in this build; use -tags playwright")
}

func sessionClickText(s *FirefoxSession, text string) error {
	return errors.New("playwright runtime is disabled in this build; use -tags playwright")
}

func sessionLoadLazyContent(s *FirefoxSession) error {
	return sessionLoadLazyContentForCount(s, 0)
}

func sessionLoadLazyContentForCount(s *FirefoxSession, expectedImageCount int) error {
	return errors.New("playwright runtime is disabled in this build; use -tags playwright")
}
