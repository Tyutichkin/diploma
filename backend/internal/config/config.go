package config

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr       string
	DBDSN          string
	JWTSecret      string
	AccessTTLMin   int
	RefreshTTLDays int
	CORSOrigins    []string
	OSRMBaseURL    string // base URL for the OSRM routing API
}

func MustLoad() Config {
	if err := loadDotEnv(".env"); err != nil {
		log.Fatal(err)
	}

	get := func(k, def string) string {
		v := strings.TrimSpace(os.Getenv(k))
		if v == "" {
			return def
		}
		return v
	}

	atoi := func(s string, def int) int {
		v, err := strconv.Atoi(s)
		if err != nil {
			return def
		}
		return v
	}

	originsRaw := get("APP_CORS_ORIGINS", "http://localhost:5173")
	origins := make([]string, 0, len(strings.Split(originsRaw, ",")))
	for _, x := range strings.Split(originsRaw, ",") {
		x = strings.TrimSpace(x)
		if x != "" {
			origins = append(origins, x)
		}
	}

	cfg := Config{
		HTTPAddr:       get("APP_HTTP_ADDR", ":8080"),
		DBDSN:          get("APP_DB_DSN", ""),
		JWTSecret:      get("APP_JWT_SECRET", ""),
		AccessTTLMin:   atoi(get("APP_ACCESS_TTL_MIN", "15"), 15),
		RefreshTTLDays: atoi(get("APP_REFRESH_TTL_DAYS", "30"), 30),
		CORSOrigins:    origins,
		OSRMBaseURL:    get("APP_OSRM_BASE_URL", "http://router.project-osrm.org"),
	}

	if cfg.DBDSN == "" || cfg.JWTSecret == "" {
		log.Fatal("APP_DB_DSN и APP_JWT_SECRET обязательны")
	}

	return cfg
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)

		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}

	return scanner.Err()
}
