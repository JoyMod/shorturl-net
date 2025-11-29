package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

// 主配置结构 - 简化命名
type Config struct {
	App       App    `yaml:"app"`
	Server    Server `yaml:"server"`
	Database  DB     `yaml:"database"`
	Cache     Cache  `yaml:"cache"`
	Auth      Auth   `yaml:"auth"`
	RateLimit Limit  `yaml:"rate_limit"`
}

// 应用配置
type App struct {
	Name    string `yaml:"name"`
	Mode    string `yaml:"mode"`
	Version string `yaml:"version"`
}

// 服务器配置
type Server struct {
	Port         int `yaml:"port"`
	ReadTimeout  int `yaml:"read_timeout"`
	WriteTimeout int `yaml:"write_timeout"`
}

// 数据库配置
type DB struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Name     string `yaml:"name"`
	Charset  string `yaml:"charset"`
}

// 缓存配置（Redis）
type Cache struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// 认证配置
type Auth struct {
	Secret          string `yaml:"secret"`
	Issuer          string `yaml:"issuer"`
	ExpirationHours int    `yaml:"expiration_hours"`
}

// 限流配置
type Limit struct {
	Enabled   bool     `yaml:"enabled"`
	Requests  int64    `yaml:"requests_per_minute"`
	Burst     int64    `yaml:"burst"`
	SkipPaths []string `yaml:"skip_paths"`
}

// 加载配置
func Load(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
