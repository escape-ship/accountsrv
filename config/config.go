package config

import (
	"log/slog"
	"os"

	"github.com/spf13/viper"
)

type (
	Config struct {
		Database Database `mapstructure:"database"`
		Auth     Auth     `mapstructure:"auth"` // 인증 관련 설정
	}

	Database struct {
		Host         string `mapstructure:"host"`          // DATABASE_HOST
		Port         int    `mapstructure:"port"`          // DATABASE_PORT
		User         string `mapstructure:"user"`          // DATABASE_USER
		Password     string `mapstructure:"password"`      // DATABASE_PASSWORD
		DataBaseName string `mapstructure:"database_name"` // DATABASE_DATABASE_NAME
		SchemaName   string `mapstructure:"schema_name"`   // DATABASE_SCHEMA_NAME
		SSLMode      string `mapstructure:"ssl_mode"`      // DATABASE_SSL_MODE
	}

	Auth struct {
		JWTSecret string `mapstructure:"jwt_secret"`
	}
)

func New(path string) (*Config, error) {
	vp := viper.New()
	vp.SetConfigFile(path)
	vp.AutomaticEnv()

	dir, err := os.Getwd()
	if err != nil {
		slog.Error("App: get current directory error", "error", err)
		os.Exit(1)
	}
	slog.Info("App: current directory", "dir", dir)

	if err := vp.ReadInConfig(); err != nil {
		return nil, err
	}
	var cfg Config
	if err := vp.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	
	// JWT Secret 환경변수에서 직접 읽기
	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		cfg.Auth.JWTSecret = jwtSecret
	}
	
	return &cfg, nil
}
