package config_test

import (
	"testing"

	"github.com/kevinball/ares-bib-logger/backend/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("MQTT_GATEWAY_NODE_ID", "a3b4c5d6")
}

func TestLoad_Defaults(t *testing.T) {
	setRequired(t)

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, 8080, cfg.ServerPort)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "localhost", cfg.DB.Host)
	assert.Equal(t, 5432, cfg.DB.Port)
	assert.Equal(t, "ares_bib_logger", cfg.DB.Name)
	assert.Equal(t, "postgres", cfg.DB.User)
	assert.Equal(t, "disable", cfg.DB.SSLMode)
	assert.True(t, cfg.MQTT.Enabled)
	assert.Equal(t, "localhost", cfg.MQTT.Host)
	assert.Equal(t, 1883, cfg.MQTT.Port)
	assert.Equal(t, "US", cfg.MQTT.Region)
	assert.Equal(t, 2, cfg.MQTT.ChannelNum)
	assert.Equal(t, "LongFast", cfg.MQTT.ChannelName)
}

func TestLoad_Overrides(t *testing.T) {
	setRequired(t)
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("DB_HOST", "db-host")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_NAME", "mydb")
	t.Setenv("DB_USER", "myuser")
	t.Setenv("DB_SSL_MODE", "require")
	t.Setenv("MQTT_HOST", "mqtt-host")
	t.Setenv("MQTT_PORT", "1884")
	t.Setenv("MQTT_REGION", "EU")
	t.Setenv("MQTT_CHANNEL_NUM", "3")
	t.Setenv("MQTT_CHANNEL_NAME", "MediumSlow")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, 9090, cfg.ServerPort)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "db-host", cfg.DB.Host)
	assert.Equal(t, 5433, cfg.DB.Port)
	assert.Equal(t, "mydb", cfg.DB.Name)
	assert.Equal(t, "myuser", cfg.DB.User)
	assert.Equal(t, "require", cfg.DB.SSLMode)
	assert.Equal(t, "mqtt-host", cfg.MQTT.Host)
	assert.Equal(t, 1884, cfg.MQTT.Port)
	assert.Equal(t, "EU", cfg.MQTT.Region)
	assert.Equal(t, 3, cfg.MQTT.ChannelNum)
	assert.Equal(t, "MediumSlow", cfg.MQTT.ChannelName)
}

func TestLoad_MQTTDisabled_NoGatewayRequired(t *testing.T) {
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("MQTT_ENABLED", "false")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.False(t, cfg.MQTT.Enabled)
}

func TestLoad_MissingDBPassword(t *testing.T) {
	t.Setenv("MQTT_ENABLED", "false")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_PASSWORD")
}

func TestLoad_MissingGatewayNodeID_WhenMQTTEnabled(t *testing.T) {
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("MQTT_ENABLED", "true")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MQTT_GATEWAY_NODE_ID")
}

func TestLoad_InvalidServerPort(t *testing.T) {
	setRequired(t)
	t.Setenv("SERVER_PORT", "not-a-number")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SERVER_PORT")
}

func TestLoad_InvalidMQTTEnabled(t *testing.T) {
	setRequired(t)
	t.Setenv("MQTT_ENABLED", "maybe")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MQTT_ENABLED")
}

func TestDBConfig_DSN(t *testing.T) {
	cfg := config.DBConfig{
		Host:     "localhost",
		Port:     5432,
		Name:     "mydb",
		User:     "myuser",
		Password: "mypass",
		SSLMode:  "disable",
	}
	assert.Equal(t,
		"host=localhost port=5432 dbname=mydb user=myuser password=mypass sslmode=disable",
		cfg.DSN(),
	)
}

func TestMQTTConfig_Topics(t *testing.T) {
	cfg := config.MQTTConfig{
		Region:        "US",
		ChannelNum:    2,
		ChannelName:   "LongFast",
		GatewayNodeID: "a3b4c5d6",
	}

	assert.Equal(t, "msh/US/2/e/LongFast/#", cfg.SubscribeTopic())
	assert.Equal(t, "msh/US/2/e/LongFast/!a3b4c5d6", cfg.PublishTopic())
}
