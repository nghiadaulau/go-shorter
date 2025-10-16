package config

import (
	"time"
)

// AppConfig holds application configuration loaded from YAML/env.
// Keep fields exported for viper to unmarshal.
type AppConfig struct {
	Server struct {
		Port         int           `mapstructure:"port"`
		ReadTimeout  time.Duration `mapstructure:"readTimeout"`
		WriteTimeout time.Duration `mapstructure:"writeTimeout"`
	} `mapstructure:"server"`
	DB struct {
		DSN          string `mapstructure:"dsn"`
		MaxOpenConns int    `mapstructure:"maxOpenConns"`
		MaxIdleConns int    `mapstructure:"maxIdleConns"`
	} `mapstructure:"db"`
	Redis struct {
		Addr string `mapstructure:"addr"`
		DB   int    `mapstructure:"db"`
		Pool int    `mapstructure:"pool"`
	} `mapstructure:"redis"`
	JWT struct {
		Secret string        `mapstructure:"secret"`
		Issuer string        `mapstructure:"issuer"`
		TTL    time.Duration `mapstructure:"ttl"`
	} `mapstructure:"jwt"`
	Redirect struct {
		CacheTTL    time.Duration `mapstructure:"cacheTTL"`
		NegativeTTL time.Duration `mapstructure:"negativeTTL"`
	} `mapstructure:"redirect"`
	RateLimit struct {
		WriteRPM int `mapstructure:"writeRPM"`
	} `mapstructure:"rateLimit"`
	Telemetry struct {
		OTLPEndpoint string `mapstructure:"otlpEndpoint"`
		Enable       bool   `mapstructure:"enable"`
	} `mapstructure:"telemetry"`
}
