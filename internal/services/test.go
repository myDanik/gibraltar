package services

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	models "gibraltar/internal/models"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

type URLTestService struct {
	URL string
}

func NewVlessTestService(url string) *URLTestService {
	return &URLTestService{
		URL: url,
	}
}

func (t *URLTestService) Test(vlessURL string, localPort int) (time.Duration, error) {
	vlessCfg, err := parseVlessURL(vlessURL)
	if err != nil {
		return 0 * time.Millisecond, err
	}
	outbound := buildVlessOutbound(*vlessCfg)
	config := buildSingBoxConfig(outbound, localPort)
	pattern := fmt.Sprintf("singbox-config-*-%s.json", vlessCfg.Server)
	tmp, err := os.CreateTemp("", pattern)
	if err != nil {
		return 0 * time.Millisecond, fmt.Errorf("create temp config: %w", err)
	}
	defer os.Remove(tmp.Name())
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(config); err != nil {
		tmp.Close()
		return 0, fmt.Errorf("write config: %w", err)
	}
	tmp.Close()
	cmd := exec.Command("sing-box", "run", "-c", tmp.Name())

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start sing-box: %w", err)
	}
	started := true
	defer func() {
		if started && cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()

	stringPort := strconv.Itoa(localPort)

	if err := waitPort(net.JoinHostPort("127.0.0.1", stringPort), 5*time.Second); err != nil {
		return 0, fmt.Errorf("sing-box inbound not ready: %w", err)
	}

	dialer, err := proxy.SOCKS5("tcp", net.JoinHostPort("127.0.0.1", stringPort), nil, proxy.Direct)
	if err != nil {
		return 0, fmt.Errorf("create socks5 dialer: %w", err)
	}

	transport := &http.Transport{
		Dial: dialer.Dial,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   2 * time.Second,
	}

	start := time.Now()
	resp, err := client.Get(t.URL)
	latency := time.Since(start)
	if err != nil {
		return 0, fmt.Errorf("probe failed: %w", err)
	}
	_ = resp.Body.Close()

	started = false
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}

	return latency, nil
}

func TLSTest(address, port, serverName string, timeout time.Duration) (time.Duration, error) {
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	start := time.Now()

	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(address, port), &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: true,
	})
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	return time.Since(start), nil
}

func parseVlessURL(raw string) (*models.VlessURL, error) {
	raw = strings.ReplaceAll(raw, "&amp;", "&")

	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "vless" {
		return nil, errors.New("not vless url")
	}

	port, _ := strconv.Atoi(u.Port())
	q := u.Query()

	return &models.VlessURL{
		UUID:        u.User.Username(),
		Server:      u.Hostname(),
		Port:        port,
		Security:    q.Get("security"),
		SNI:         q.Get("sni"),
		PublicKey:   q.Get("pbk"),
		SID:         q.Get("sid"),
		Fingerprint: q.Get("fp"),
		Type:        q.Get("type"),
		SPX:         q.Get("spx"),
		Flow:        q.Get("flow"),
		Path:        q.Get("path"),
		Host:        q.Get("host"),
		ServiceName: q.Get("serviceName"),
		HeaderType:  q.Get("headerType"),
	}, nil
}

func buildVlessOutbound(v models.VlessURL) map[string]any {
	outbound := map[string]any{
		"type":        "vless",
		"tag":         "proxy",
		"server":      v.Server,
		"server_port": v.Port,
		"uuid":        v.UUID,
	}

	if v.Flow != "" {
		outbound["flow"] = v.Flow
	}

	outbound["network"] = mapNetwork(v.Type)

	switch strings.ToLower(v.Security) {
	case "reality":
		outbound["tls"] = map[string]any{
			"enabled":     true,
			"server_name": v.SNI,
			"utls": map[string]any{
				"enabled": true,
			},
			"reality": map[string]any{
				"enabled":    true,
				"public_key": v.PublicKey,
				"short_id":   v.SID,
			},
		}
	case "tls", "ssl":
		outbound["tls"] = map[string]any{
			"enabled":     true,
			"server_name": v.SNI,
		}
	}

	return outbound
}

func mapNetwork(t string) string {
	switch strings.ToLower(t) {
	case "", "tcp", "raw":
		return "tcp"
	case "udp", "quic":
		return "udp"
	default:
		return "tcp"
	}
}

func buildTransport(v models.VlessURL) map[string]any {
	switch strings.ToLower(v.Type) {
	case "ws":
		ws := map[string]any{"type": "ws"}
		if v.Path != "" {
			ws["path"] = v.Path
		}
		if v.Host != "" {
			ws["headers"] = map[string]any{"Host": v.Host}
		} else if v.SNI != "" {
			ws["headers"] = map[string]any{"Host": v.SNI}
		}
		return ws
	case "grpc":
		grpc := map[string]any{"type": "grpc"}
		if v.ServiceName != "" {
			grpc["service_name"] = v.ServiceName
		}
		return grpc
	case "http", "h2", "http2", "xhttp":
		httpT := map[string]any{"type": "http"}
		if v.Path != "" {
			httpT["path"] = v.Path
		}
		if v.Host != "" {
			httpT["headers"] = map[string]any{"Host": v.Host}
		} else if v.SNI != "" {
			httpT["headers"] = map[string]any{"Host": v.SNI}
		}
		return httpT
	default:
		return nil
	}
}

func buildSingBoxConfig(outbound map[string]any, port int) map[string]any {
	tag := "proxy"
	if t, ok := outbound["tag"].(string); ok && t != "" {
		tag = t
	}
	return map[string]any{
		"log": map[string]any{"level": "error"},
		"inbounds": []any{
			map[string]any{
				"type":        "socks",
				"tag":         "socks-in",
				"listen":      "127.0.0.1",
				"listen_port": port,
			},
		},
		"outbounds": []any{
			outbound,
			map[string]any{"type": "direct", "tag": "direct"},
		},
		"route": map[string]any{
			"rules": []any{
				map[string]any{"outbound": tag},
			},
		},
	}
}

func waitPort(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 400*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", addr)
}
