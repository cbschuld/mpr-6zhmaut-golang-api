package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

var ValidBaudRates = []int{9600, 19200, 38400, 57600, 115200, 230400}

type Config struct {
	Device         string
	TargetBaudRate int
	Port           int
	AmpCount       int
	CORS           bool
	PollInterval   time.Duration
	HealthInterval time.Duration
	CmdTimeout     time.Duration
	StepDelay      time.Duration
	LogLevel       string
}

func Load() (*Config, error) {
	cfg := &Config{
		Device:         envStr("DEVICE", "/dev/ttyUSB0"),
		TargetBaudRate: envInt("TARGET_BAUDRATE", 115200),
		Port:           envInt("PORT", 8181),
		AmpCount:       envInt("AMPCOUNT", 1),
		CORS:           envBool("CORS", false),
		PollInterval:   envDuration("POLL_INTERVAL", 5*time.Second),
		HealthInterval: envDuration("HEALTH_INTERVAL", 30*time.Second),
		CmdTimeout:     envDuration("CMD_TIMEOUT", 2*time.Second),
		StepDelay:      envDuration("STEP_DELAY", 500*time.Millisecond),
		LogLevel:       envStr("LOG_LEVEL", "info"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.AmpCount < 1 || c.AmpCount > 3 {
		return fmt.Errorf("AMPCOUNT must be 1, 2, or 3 (got %d)", c.AmpCount)
	}
	if !isValidBaudRate(c.TargetBaudRate) {
		return fmt.Errorf("TARGET_BAUDRATE %d is not valid, use one of: %v", c.TargetBaudRate, ValidBaudRates)
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("PORT must be 1-65535 (got %d)", c.Port)
	}
	return nil
}

func isValidBaudRate(rate int) bool {
	for _, v := range ValidBaudRates {
		if v == rate {
			return true
		}
	}
	return false
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
