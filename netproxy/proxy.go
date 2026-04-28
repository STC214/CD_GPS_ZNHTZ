package netproxy

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// NormalizeServer cleans and validates a user supplied proxy server.
func NormalizeServer(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse proxy server: %w", err)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "socks4", "socks4a", "socks5", "socks5h":
	default:
		return "", fmt.Errorf("unsupported proxy scheme %q", parsed.Scheme)
	}
	if parsed.Hostname() == "" {
		return "", fmt.Errorf("proxy host is empty")
	}
	if parsed.Port() == "" {
		return "", fmt.Errorf("proxy port is empty")
	}
	return parsed.String(), nil
}

// NewHTTPClient returns an HTTP client that uses the supplied proxy server.
// Empty proxyServer keeps the standard environment/system proxy behavior.
func NewHTTPClient(proxyServer string, timeout time.Duration) (*http.Client, error) {
	transport, err := NewTransport(proxyServer)
	if err != nil {
		return nil, err
	}
	return &http.Client{Transport: transport, Timeout: timeout}, nil
}

// NewTransport returns a transport for HTTP image/API downloads.
func NewTransport(proxyServer string) (*http.Transport, error) {
	proxyServer, err := NormalizeServer(proxyServer)
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if proxyServer == "" {
		transport.Proxy = http.ProxyFromEnvironment
		return transport, nil
	}
	proxyURL, err := url.Parse(proxyServer)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(proxyURL.Scheme) {
	case "http", "https", "socks5", "socks5h":
		transport.Proxy = http.ProxyURL(proxyURL)
	case "socks4", "socks4a":
		transport.Proxy = nil
		transport.DialContext = socks4DialContext(proxyURL)
	default:
		return nil, fmt.Errorf("unsupported proxy scheme %q", proxyURL.Scheme)
	}
	return transport, nil
}

func socks4DialContext(proxyURL *url.URL) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if network != "tcp" && network != "tcp4" && network != "tcp6" {
			return nil, fmt.Errorf("unsupported socks4 network %q", network)
		}
		var d net.Dialer
		conn, err := d.DialContext(ctx, "tcp", proxyURL.Host)
		if err != nil {
			return nil, err
		}
		if err := socks4Handshake(conn, strings.EqualFold(proxyURL.Scheme, "socks4a"), address); err != nil {
			_ = conn.Close()
			return nil, err
		}
		return conn, nil
	}
}

func socks4Handshake(conn net.Conn, socks4a bool, address string) error {
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 || port > 65535 {
		return fmt.Errorf("invalid target port %q", portText)
	}
	ip := net.ParseIP(host).To4()
	useSocks4a := socks4a || ip == nil
	if ip == nil {
		ip = net.IPv4(0, 0, 0, 1)
	}
	req := []byte{0x04, 0x01, 0, 0, ip[0], ip[1], ip[2], ip[3], 0}
	binary.BigEndian.PutUint16(req[2:4], uint16(port))
	if useSocks4a {
		req = append(req, []byte(host)...)
		req = append(req, 0)
	}
	if _, err := conn.Write(req); err != nil {
		return err
	}
	resp := make([]byte, 8)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return err
	}
	if resp[1] != 0x5a {
		return fmt.Errorf("socks4 proxy rejected request with status 0x%02x", resp[1])
	}
	return nil
}
