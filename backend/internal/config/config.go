package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ServerPort int
	LogLevel   string
	Timezone   string
	DB         DBConfig
	MQTT       MQTTConfig
}

type DBConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

// DSN returns a lib/pq compatible connection string.
func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		c.Host, c.Port, c.Name, c.User, c.Password, c.SSLMode,
	)
}

type MQTTConfig struct {
	Enabled       bool
	Host          string
	Port          int
	Region        string
	ChannelNum    int
	ChannelName   string
	GatewayNodeID string
}

// SubscribeTopic returns the wildcard topic to receive all mesh traffic on the channel.
func (c MQTTConfig) SubscribeTopic() string {
	return fmt.Sprintf("msh/%s/%d/e/%s/#", c.Region, c.ChannelNum, c.ChannelName)
}

// PublishTopic returns the topic used to send messages back through the gateway to the mesh.
func (c MQTTConfig) PublishTopic() string {
	return fmt.Sprintf("msh/%s/%d/e/%s/!%s", c.Region, c.ChannelNum, c.ChannelName, c.GatewayNodeID)
}

// Load reads all configuration from environment variables and returns a validated Config.
func Load() (*Config, error) {
	serverPort, err := envInt("SERVER_PORT", 8080)
	if err != nil {
		return nil, fmt.Errorf("SERVER_PORT: %w", err)
	}

	dbPort, err := envInt("DB_PORT", 5432)
	if err != nil {
		return nil, fmt.Errorf("DB_PORT: %w", err)
	}

	mqttEnabled, err := envBool("MQTT_ENABLED", true)
	if err != nil {
		return nil, fmt.Errorf("MQTT_ENABLED: %w", err)
	}

	mqttPort, err := envInt("MQTT_PORT", 1883)
	if err != nil {
		return nil, fmt.Errorf("MQTT_PORT: %w", err)
	}

	mqttChannelNum, err := envInt("MQTT_CHANNEL_NUM", 2)
	if err != nil {
		return nil, fmt.Errorf("MQTT_CHANNEL_NUM: %w", err)
	}

	cfg := &Config{
		ServerPort: serverPort,
		LogLevel:   envStr("LOG_LEVEL", "info"),
		Timezone:   envStr("TIMEZONE", "Local"),
		DB: DBConfig{
			Host:     envStr("DB_HOST", "localhost"),
			Port:     dbPort,
			Name:     envStr("DB_NAME", "ares_bib_logger"),
			User:     envStr("DB_USER", "postgres"),
			Password: envStr("DB_PASSWORD", ""),
			SSLMode:  envStr("DB_SSL_MODE", "disable"),
		},
		MQTT: MQTTConfig{
			Enabled:       mqttEnabled,
			Host:          envStr("MQTT_HOST", "localhost"),
			Port:          mqttPort,
			Region:        envStr("MQTT_REGION", "US"),
			ChannelNum:    mqttChannelNum,
			ChannelName:   envStr("MQTT_CHANNEL_NAME", "LongFast"),
			GatewayNodeID: envStr("MQTT_GATEWAY_NODE_ID", ""),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	var missing []string

	if c.DB.Password == "" {
		missing = append(missing, "DB_PASSWORD")
	}
	if c.MQTT.Enabled && c.MQTT.GatewayNodeID == "" {
		missing = append(missing, "MQTT_GATEWAY_NODE_ID (required when MQTT_ENABLED=true)")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

func envStr(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envInt(key string, defaultVal int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q", v)
	}
	return n, nil
}

func envBool(key string, defaultVal bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("invalid boolean %q", v)
	}
	return b, nil
}
