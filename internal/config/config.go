package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Listen    string           `json:"listen"`
	AllowLAN  bool             `json:"allow_lan"`
	Providers []ProviderConfig `json:"providers"`
	Routing   RoutingConfig    `json:"routing,omitempty"`
	Routes    []RouteConfig    `json:"routes,omitempty"`
}

type RoutingConfig struct {
	RouteAliasesEnabled bool                    `json:"route_aliases_enabled,omitempty"`
	SameModelFallback   SameModelFallbackConfig `json:"same_model_fallback,omitempty"`
}

type SameModelFallbackConfig struct {
	Enabled   bool     `json:"enabled,omitempty"`
	Mode      string   `json:"mode,omitempty"`
	Providers []string `json:"providers,omitempty"`
}

type RouteConfig struct {
	ID      string   `json:"id"`
	Targets []string `json:"targets"`
}

type ProviderConfig struct {
	ID                    string            `json:"id"`
	DisplayName           string            `json:"display_name,omitempty"`
	Description           string            `json:"description,omitempty"`
	Enabled               bool              `json:"enabled"`
	Kind                  string            `json:"kind"`
	BaseURL               string            `json:"base_url"`
	UIURL                 string            `json:"ui_url,omitempty"`
	DocsURL               string            `json:"docs_url,omitempty"`
	Binary                string            `json:"binary,omitempty"`
	Args                  []string          `json:"args,omitempty"`
	WorkDir               string            `json:"work_dir,omitempty"`
	Env                   map[string]string `json:"env,omitempty"`
	Headers               map[string]string `json:"headers,omitempty"`
	HealthPath            string            `json:"health_path,omitempty"`
	ModelsPath            string            `json:"models_path,omitempty"`
	MaxRestarts           int               `json:"max_restarts,omitempty"`
	StartupTimeoutSeconds int               `json:"startup_timeout_seconds,omitempty"`
	HealthIntervalSeconds int               `json:"health_interval_seconds,omitempty"`
}

func Load(path string) (Config, error) {
	if strings.TrimSpace(path) == "" {
		path = "config.json"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Listen == "" {
		cfg.Listen = "127.0.0.1:8317"
	}
	if strings.TrimSpace(cfg.Routing.SameModelFallback.Mode) == "" {
		cfg.Routing.SameModelFallback.Mode = "sequential"
	}
	root, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return Config{}, fmt.Errorf("resolve config directory: %w", err)
	}
	for i := range cfg.Providers {
		p := &cfg.Providers[i]
		p.ID = strings.TrimSpace(p.ID)
		if strings.TrimSpace(p.DisplayName) == "" {
			p.DisplayName = p.ID
		}
		if p.HealthPath == "" {
			p.HealthPath = "/healthz"
		}
		if p.ModelsPath == "" {
			p.ModelsPath = "/v1/models"
		}
		if p.MaxRestarts <= 0 {
			p.MaxRestarts = 3
		}
		if p.StartupTimeoutSeconds <= 0 {
			p.StartupTimeoutSeconds = 30
		}
		if p.HealthIntervalSeconds <= 0 {
			p.HealthIntervalSeconds = 15
		}
		if p.Binary != "" && !filepath.IsAbs(p.Binary) {
			p.Binary = filepath.Join(root, p.Binary)
		}
		if p.WorkDir != "" && !filepath.IsAbs(p.WorkDir) {
			p.WorkDir = filepath.Join(root, p.WorkDir)
		}
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if err := validateListen(c.Listen, c.AllowLAN); err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for _, p := range c.Providers {
		id := strings.TrimSpace(p.ID)
		if id == "" {
			return errors.New("provider id is required")
		}
		if strings.Contains(id, "/") {
			return fmt.Errorf("provider %q: id must not contain '/'", id)
		}
		if id == "route" || id == "pool" {
			return fmt.Errorf("provider id %q is reserved for AhB virtual routing", id)
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("duplicate provider id %q", id)
		}
		seen[id] = struct{}{}

		if p.UIURL != "" {
			if err := validatePublicURL("ui_url", id, p.UIURL); err != nil {
				return err
			}
		}
		if p.DocsURL != "" {
			if err := validatePublicURL("docs_url", id, p.DocsURL); err != nil {
				return err
			}
		}

		if !p.Enabled {
			continue
		}
		if p.Kind != "sidecar" {
			return fmt.Errorf("provider %q: V1 only supports kind=sidecar", id)
		}
		if strings.TrimSpace(p.Binary) == "" {
			return fmt.Errorf("provider %q: binary is required", id)
		}
		if strings.TrimSpace(p.BaseURL) == "" {
			return fmt.Errorf("provider %q: base_url is required", id)
		}
		u, err := url.Parse(p.BaseURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("provider %q: invalid base_url", id)
		}
		if u.User != nil {
			return fmt.Errorf("provider %q: base_url must not contain credentials", id)
		}
		host := u.Hostname()
		ip := net.ParseIP(host)
		if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return fmt.Errorf("provider %q: sidecar base_url must be loopback in V1", id)
		}
	}

	mode := strings.ToLower(strings.TrimSpace(c.Routing.SameModelFallback.Mode))
	if mode == "" {
		mode = "sequential"
	}
	if mode != "sequential" && mode != "parallel" {
		return fmt.Errorf("same_model_fallback mode must be sequential or parallel")
	}
	fallbackSeen := map[string]struct{}{}
	for _, rawID := range c.Routing.SameModelFallback.Providers {
		id := strings.TrimSpace(rawID)
		if id == "" {
			return fmt.Errorf("same_model_fallback provider id must not be empty")
		}
		if _, ok := seen[id]; !ok {
			return fmt.Errorf("same_model_fallback provider %q does not exist", id)
		}
		if _, ok := fallbackSeen[id]; ok {
			return fmt.Errorf("duplicate same_model_fallback provider %q", id)
		}
		fallbackSeen[id] = struct{}{}
	}

	routeSeen := map[string]struct{}{}
	for _, route := range c.Routes {
		id := strings.TrimSpace(route.ID)
		if id == "" {
			return errors.New("route id is required")
		}
		if strings.Contains(id, "/") {
			return fmt.Errorf("route %q: id must not contain '/'", id)
		}
		if _, ok := routeSeen[id]; ok {
			return fmt.Errorf("duplicate route id %q", id)
		}
		routeSeen[id] = struct{}{}
		if len(route.Targets) == 0 {
			return fmt.Errorf("route %q: at least one target is required", id)
		}
		targetSeen := map[string]struct{}{}
		for _, rawTarget := range route.Targets {
			target := strings.TrimSpace(rawTarget)
			providerID, modelID, ok := strings.Cut(target, "/")
			if !ok || providerID == "" || modelID == "" {
				return fmt.Errorf("route %q: target %q must use provider/model format", id, rawTarget)
			}
			if providerID == "route" {
				return fmt.Errorf("route %q: nested route targets are not supported", id)
			}
			if _, ok := seen[providerID]; !ok {
				return fmt.Errorf("route %q: target provider %q does not exist", id, providerID)
			}
			if _, ok := targetSeen[target]; ok {
				return fmt.Errorf("route %q: duplicate target %q", id, target)
			}
			targetSeen[target] = struct{}{}
		}
	}
	return nil
}

func validatePublicURL(field, id, raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("provider %q: invalid %s", id, field)
	}
	if u.User != nil {
		return fmt.Errorf("provider %q: %s must not contain credentials", id, field)
	}
	return nil
}

func validateListen(addr string, allowLAN bool) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid listen address %q: %w", addr, err)
	}
	if allowLAN {
		return fmt.Errorf("allow_lan is reserved for a later authenticated release; V1 is loopback-only")
	}
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("listen address %q is not loopback; V1 is intentionally loopback-only", addr)
	}
	return nil
}