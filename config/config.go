package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type (
	Config struct {
		App      App      `mapstructure:"app"`
		Log      Log      `mapstructure:"log"`
		HTTP     HTTP     `mapstructure:"http"`
		Postgres Postgres `mapstructure:"postgres"`
		Redis    Redis    `mapstructure:"redis"`
		MinIO    MinIO    `mapstructure:"minio"`
		File     File     `mapstructure:"file_storage"`
		Captcha  Captcha  `mapstructure:"captcha"`
		SMTP     SMTP     `mapstructure:"smtp"`
		JWT      JWT      `mapstructure:"jwt"`
		TwoFA    TwoFA    `mapstructure:"twofa"`
		OpenAI   OpenAI   `mapstructure:"openai"`
		Metrics  Metrics  `mapstructure:"metrics"`
		Swagger  Swagger  `mapstructure:"swagger"`
	}

	App struct {
		Name    string `mapstructure:"name"`
		Version string `mapstructure:"version"`
	}

	Log struct {
		Level string `mapstructure:"level"`
	}

	HTTP struct {
		Port           string `mapstructure:"port"`
		UsePreforkMode bool   `mapstructure:"use_prefork_mode"`
	}

	Postgres struct {
		Host         string `mapstructure:"host"`
		Port         int    `mapstructure:"port"`
		User         string `mapstructure:"user"`
		Password     string `mapstructure:"password"`
		DBName       string `mapstructure:"dbname"`
		SSLMode      string `mapstructure:"sslmode"`
		TimeZone     string `mapstructure:"time_zone"`
		MaxIdleConns int    `mapstructure:"max_idle_conns"`
		MaxOpenConns int    `mapstructure:"max_open_conns"`
	}

	Redis struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
	}

	MinIO struct {
		Endpoint  string `mapstructure:"endpoint"`
		AccessKey string `mapstructure:"access_key"`
		SecretKey string `mapstructure:"secret_key"`
		UseSSL    bool   `mapstructure:"use_ssl"`
		Bucket    string `mapstructure:"bucket"`
		Region    string `mapstructure:"region"`
	}

	File struct {
		Provider      string `mapstructure:"provider"`
		PublicBaseURL string `mapstructure:"public_base_url"`
	}

	Captcha struct {
		Height   int     `mapstructure:"height"`
		Width    int     `mapstructure:"width"`
		Length   int     `mapstructure:"length"`
		MaxSkew  float64 `mapstructure:"max_skew"`
		DotCount int     `mapstructure:"dot_count"`
	}

	SMTP struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
		From     string `mapstructure:"from"`
	}

	JWT struct {
		AccessSecret  string        `mapstructure:"access_secret"`
		RefreshSecret string        `mapstructure:"refresh_secret"`
		AccessTTL     time.Duration `mapstructure:"access_ttl"`
		RefreshTTL    time.Duration `mapstructure:"refresh_ttl"`
		Issuer        string        `mapstructure:"issuer"`
	}

	TwoFA struct {
		QRWidth       int    `mapstructure:"qr_width"`
		QRHeight      int    `mapstructure:"qr_height"`
		EncryptionKey string `mapstructure:"encryption_key"`
	}

	OpenAI struct {
		APIKey  string `mapstructure:"api_key"`
		BaseURL string `mapstructure:"base_url"`
		Model   string `mapstructure:"model"`
	}

	Metrics struct {
		Enabled bool `mapstructure:"enabled"`
	}

	Swagger struct {
		Enabled bool `mapstructure:"enabled"`
	}
)

func NewConfig() (*Config, error) {
	cfg := &Config{}
	viper.SetConfigFile("config.yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	return cfg, nil
}
