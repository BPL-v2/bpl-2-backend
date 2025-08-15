package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

// Config holds all environment configuration
type Config struct {
	// Database
	DatabaseHost     string
	DatabasePort     string
	PostgresUser     string
	PostgresPassword string
	DatabaseName     string

	// Authentication
	JWTSecret string

	// Discord
	DiscordClientID     string
	DiscordClientSecret string
	DiscordGuildID      string
	DiscordBotToken     string
	DiscordBotURL       string

	// Twitch
	TwitchClientID     string
	TwitchClientSecret string

	// Path of Exile
	POEClientID     string
	POEClientSecret string
	POEClientAgent  string
	OldPOEToken     string

	// Path of Building
	POBServerURL string

	// Other
	KafkaBroker string
}

var (
	appConfig *Config
	onceEnv   sync.Once
)

// LoadConfig loads and validates all environment variables
func loadConfig() *Config {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		// Database - required
		DatabaseHost:     getEnvWithDefault("DATABASE_HOST", "localhost"),
		DatabasePort:     getEnvWithDefault("DATABASE_PORT", "5432"),
		PostgresUser:     getEnvWithDefault("POSTGRES_USER", "postgres"),
		PostgresPassword: getEnvWithDefault("POSTGRES_PASSWORD", "postgres"),
		DatabaseName:     getEnvWithDefault("DATABASE_NAME", "postgres"),

		// JWT - required
		JWTSecret: getEnvWithDefault("JWT_SECRET", "dummyjwt"),

		// Discord - optional
		DiscordClientID:     getEnv("DISCORD_CLIENT_ID"),
		DiscordClientSecret: getEnv("DISCORD_CLIENT_SECRET"),
		DiscordGuildID:      getEnv("DISCORD_GUILD_ID"),
		DiscordBotToken:     getEnv("DISCORD_BOT_TOKEN"),
		DiscordBotURL:       getEnv("DISCORD_BOT_URL"),

		// Twitch - optional
		TwitchClientID:     getEnv("TWITCH_CLIENT_ID"),
		TwitchClientSecret: getEnv("TWITCH_CLIENT_SECRET"),

		// Path of Exile - required for game functionality
		POEClientID:     getEnv("POE_CLIENT_ID"),
		POEClientSecret: getEnv("POE_CLIENT_SECRET"),
		POEClientAgent:  getEnv("POE_CLIENT_AGENT"),
		OldPOEToken:     getEnv("OLD_POE_CLIENT_TOKEN"),

		// Path of Building - optional
		POBServerURL: getEnvWithDefault("POB_SERVER_URL", "localhost:8080"),

		// Other
		KafkaBroker: getEnvWithDefault("KAFKA_BROKER", "localhost:9092"),
	}

	appConfig = config
	return config
}

func Env() *Config {
	onceEnv.Do(func() {
		appConfig = loadConfig()
	})
	return appConfig
}

// Helper functions
func getEnv(key string) string {
	value := os.Getenv(key)
	if value == "" && IsProduction() {
		panic(fmt.Sprintf("Required environment variable %s is not set", key))
	}
	return value
}

func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// IsProduction returns true if running in production
func IsProduction() bool {
	return getEnvWithDefault("ENVIRONMENT", "development") == "production"
}

// IsDevelopment returns true if running in development
func IsDevelopment() bool {
	return !IsProduction()
}
