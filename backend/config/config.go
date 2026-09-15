package config

import (
	"os"
	"strconv"
)

// Config 应用配置
type Config struct {
	ServerPort int
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	RedisAddr  string
}

// Load 从环境变量加载配置，未设置时使用默认值
func Load() *Config {
	return &Config{
		ServerPort: getEnvInt("SERVER_PORT", 8080),
		DBHost:     getEnv("DB_HOST", "127.0.0.1"),
		DBPort:     getEnvInt("DB_PORT", 3306),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", "root"),
		DBName:     getEnv("DB_NAME", "kpl_bp"),
		RedisAddr:  getEnv("REDIS_ADDR", "127.0.0.1:6379"),
	}
}

// DSN 返回 MySQL 连接串
func (c *Config) DSN() string {
	return c.DBUser + ":" + c.DBPassword + "@tcp(" + c.DBHost + ":" + strconv.Itoa(c.DBPort) + ")/" + c.DBName + "?charset=utf8mb4&parseTime=True&loc=Local"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return fallback
}