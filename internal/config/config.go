package config

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Radius   RadiusConfig  `yaml:"radius"`
	Database DatabaseConfig `yaml:"database"`
	Web      WebConfig     `yaml:"web"`
	Admins   []AdminUser   `yaml:"admins"`
}

type RadiusConfig struct {
	SharedSecret string `yaml:"shared_secret"`
	Port         int    `yaml:"port"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type WebConfig struct {
	Port          int    `yaml:"port"`
	SessionSecret string `yaml:"session_secret"`
}

type AdminUser struct {
	Username     string `yaml:"username"`
	PasswordHash string `yaml:"password_hash"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	cfg.applyDefaults()
	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Radius.SharedSecret == "" {
		return fmt.Errorf("radius.shared_secret is required")
	}
	if c.Web.SessionSecret == "" {
		return fmt.Errorf("web.session_secret is required")
	}
	return nil
}

func (c *Config) applyDefaults() {
	if c.Radius.Port == 0 {
		c.Radius.Port = 1812
	}
	if c.Web.Port == 0 {
		c.Web.Port = 8080
	}
	if c.Database.Path == "" {
		c.Database.Path = "./radius.db"
	}
}

func (c *Config) FindAdmin(username string) (*AdminUser, bool) {
	for i := range c.Admins {
		if c.Admins[i].Username == username {
			return &c.Admins[i], true
		}
	}
	return nil, false
}

func (a *AdminUser) CheckPassword(plain string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(plain))
	return err == nil
}
