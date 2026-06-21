package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Listen string `yaml:"listen"`
}

type AdminConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type JWTConfig struct {
	Secret string `yaml:"secret"`
	Expiry string `yaml:"expiry"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type ProxyChainConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Type     string `yaml:"type"`
}

type TelegramConfig struct {
	BotToken string `yaml:"bot_token"`
	Enabled  bool   `yaml:"enabled"`
}

type NodeConfig struct {
	Secret string `yaml:"secret"`
}

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Admin      AdminConfig      `yaml:"admin"`
	JWT        JWTConfig        `yaml:"jwt"`
	Database   DatabaseConfig   `yaml:"database"`
	ProxyChain ProxyChainConfig `yaml:"proxy_chain"`
	Telegram   TelegramConfig   `yaml:"telegram"`
	Node       NodeConfig       `yaml:"node"`
}

func (c *Config) HasNodeSecret() bool { return c.Node.Secret != "" }

func (c *Config) JWTExpiry() time.Duration {
	d, err := time.ParseDuration(c.JWT.Expiry)
	if err != nil {
		return 24 * time.Hour
	}
	return d
}

func (c *Config) HasProxyChain() bool {
	return c.ProxyChain.Host != "" && c.ProxyChain.Port > 0
}

func (c *Config) HasTelegram() bool {
	return c != nil && c.Telegram.Enabled && c.Telegram.BotToken != ""
}

var C Config

func Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(data, &C); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	return nil
}
