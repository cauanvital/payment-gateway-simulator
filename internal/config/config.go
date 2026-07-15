// Package config centraliza o carregamento da configuração da aplicação a
// partir de variáveis de ambiente, com valores padrão sensatos para
// desenvolvimento local.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

// Config agrega toda a configuração da aplicação.
type Config struct {
	Env    string
	Server ServerConfig
	DB     DBConfig
	Log    LogConfig
}

// ServerConfig contém os parâmetros do servidor HTTP.
type ServerConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// DBConfig contém os parâmetros de conexão com o Postgres.
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// LogConfig contém os parâmetros de logging estruturado (slog).
type LogConfig struct {
	Format string // "json" ou "text"
	Level  string // "debug", "info", "warn", "error"
}

// DSN monta a string de conexão do Postgres no formato URL.
func (d DBConfig) DSN() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(d.User, d.Password),
		Host:   fmt.Sprintf("%s:%s", d.Host, d.Port),
		Path:   d.Name,
	}
	q := u.Query()
	q.Set("sslmode", d.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

// Load lê a configuração do ambiente. Retorna erro se algum valor de
// duração for inválido.
func Load() (*Config, error) {
	server := ServerConfig{
		Port: getEnv("SERVER_PORT", "8080"),
	}

	var err error
	if server.ReadTimeout, err = getEnvDuration("SERVER_READ_TIMEOUT", 15*time.Second); err != nil {
		return nil, err
	}
	if server.WriteTimeout, err = getEnvDuration("SERVER_WRITE_TIMEOUT", 15*time.Second); err != nil {
		return nil, err
	}
	if server.ShutdownTimeout, err = getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 10*time.Second); err != nil {
		return nil, err
	}

	cfg := &Config{
		Env:    getEnv("APP_ENV", "development"),
		Server: server,
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "cardflow"),
			Password: getEnv("DB_PASSWORD", "cardflow"),
			Name:     getEnv("DB_NAME", "cardflow"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Log: LogConfig{
			Format: getEnv("LOG_FORMAT", "text"),
			Level:  getEnv("LOG_LEVEL", "info"),
		},
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("config: %s inválido (%q): %w", key, v, err)
	}
	return d, nil
}

// getEnvInt é um helper reservado para futuros parâmetros numéricos.
func getEnvInt(key string, fallback int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("config: %s inválido (%q): %w", key, v, err)
	}
	return n, nil
}
