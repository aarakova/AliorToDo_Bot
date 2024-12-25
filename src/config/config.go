package config

import (
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"strconv"
	"time"
)

// Config структура, которая хранит настройки приложения
type Config struct {
	Telegram TelegramConfig `yaml:"telegram"`
	UI       UIConfig       `yaml:"ui"`
	Database DBConfig       `yaml:"database"`
}

// TelegramConfig хранит параметры для Telegram API
type TelegramConfig struct {
	Token string `yaml:"token"`
	Debug bool   `yaml:"debug"`
}

// UIConfig хранит параметры для управления пользовательским интерфейсом
type UIConfig struct {
	SessionTTL      time.Duration `yaml:"session_ttl"`
	CleanerInterval time.Duration `yaml:"cleaner_interval"`
}

// DBConfig хранит параметры для подключения к базе данных
type DBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

// DSN генерирует строку подключения для базы данных
func (db *DBConfig) DSN() string {
	return "host=" + db.Host +
		" port=" + strconv.Itoa(db.Port) +
		" user=" + db.User +
		" password=" + db.Password +
		" dbname=" + db.DBName +
		" sslmode=" + db.SSLMode
}

// LoadConfig загружает конфигурацию из файла
func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &Config{}
	decoder := yaml.NewDecoder(file)
	return config, decoder.Decode(config)
}

// DefaultConfig возвращает конфигурацию с параметрами по умолчанию
func DefaultConfig() *Config {
	return &Config{
		Telegram: TelegramConfig{
			Token: "",
			Debug: false,
		},
		UI: UIConfig{
			SessionTTL:      10 * time.Minute,
			CleanerInterval: 20 * time.Second,
		},
		Database: DBConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "postgres",
			DBName:   "example",
			SSLMode:  "disable",
		},
	}
}

// ParseEnv обновляет конфигурацию из переменных окружения
func (c *Config) ParseEnv() {
	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		c.Telegram.Token = token
	}

	if debug := os.Getenv("TELEGRAM_DEBUG"); debug != "" {
		c.Telegram.Debug, _ = strconv.ParseBool(debug)
	}

	if ttl := os.Getenv("UI_SESSION_TTL"); ttl != "" {
		parsedTTL, err := time.ParseDuration(ttl)
		if err == nil {
			c.UI.SessionTTL = parsedTTL
		} else {
			log.Printf("Ошибка парсинга UI_SESSION_TTL: %v", err)
		}
	}

	if host := os.Getenv("DB_HOST"); host != "" {
		c.Database.Host = host
	}
	if port := os.Getenv("DB_PORT"); port != "" {
		c.Database.Port, _ = strconv.Atoi(port)
	}
	if user := os.Getenv("DB_USER"); user != "" {
		c.Database.User = user
	}
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		c.Database.Password = password
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		c.Database.DBName = dbName
	}
	if sslMode := os.Getenv("DB_SSLMODE"); sslMode != "" {
		c.Database.SSLMode = sslMode
	}
}
