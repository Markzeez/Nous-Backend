package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

type Role string

const (
	RolePatient      Role = "PATIENT"
	RoleDoctor       Role = "DOCTOR"
	RolePharmacist   Role = "PHARMACIST"
	RoleNurse        Role = "NURSE"
	RoleLabScientist Role = "LAB_SCIENTIST"
	RoleAdmin        Role = "ADMIN"
)

type Config struct {
	Port           string
	MongoURI       string
	MongoDB        string
	JWTSecret      string
	JWTExpiryHours int
	FrontendURL    string

	// Email/SMTP config
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPFrom string
}

func Load() *Config {
	possiblePaths := []string{
		".env",
		filepath.Join("..", ".env"),
		filepath.Join("Backend", ".env"),
	}

	if _, filename, _, ok := runtime.Caller(0); ok {
		projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
		possiblePaths = append(possiblePaths, filepath.Join(projectRoot, ".env"))
	}

	var loaded bool
	for _, p := range possiblePaths {
		if err := godotenv.Load(p); err == nil {
			log.Printf("[config] Loaded environment from %s", p)
			loaded = true
			break
		}
	}
	if !loaded {
		log.Println("[config] No .env file found; relying on process environment variables")
	}

	return &Config{
		Port:           getEnv("PORT", "8080"),
		MongoURI:       mustEnv("MONGO_URI"),
		MongoDB:        getEnv("MONGO_DB", "medcon"),
		JWTSecret:      mustEnv("JWT_SECRET"),
		JWTExpiryHours: 24, // default 24 hours
		FrontendURL:    getEnv("FRONTEND_URL", "http://localhost:5173"),
		SMTPHost:       getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:       getEnvInt("SMTP_PORT", 587),
		SMTPUser:       getEnv("SMTP_USER", ""),
		SMTPPass:       getEnv("SMTP_PASS", ""),
		SMTPFrom:       getEnv("SMTP_FROM", "noreply@medconnect.com"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var result int
		fmt.Sscanf(v, "%d", &result)
		if result > 0 {
			return result
		}
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}
