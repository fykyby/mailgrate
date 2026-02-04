package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type config struct {
	AppName                  string
	AppKey                   string
	Debug                    bool
	Port                     int
	DatabaseURL              string
	WorkerCount              int
	JobTimeoutMinutes        int
	RequireEmailConfirmation bool
	SMTPHost                 string
	SMTPPort                 int
	SMTPLogin                string
	SMTPPassword             string
}

var Config *config

func InitConfig() {
	godotenv.Load()
	var cfg = new(config)

	cfg.AppName = os.Getenv("APP_NAME")
	cfg.AppKey = os.Getenv("APP_KEY")
	cfg.Debug = os.Getenv("DEBUG") == "true"
	cfg.Port, _ = strconv.Atoi(os.Getenv("PORT"))
	cfg.DatabaseURL = os.Getenv("DB_URI")
	cfg.WorkerCount, _ = strconv.Atoi(os.Getenv("WORKER_COUNT"))
	cfg.JobTimeoutMinutes, _ = strconv.Atoi(os.Getenv("JOB_TIMEOUT_MINUTES"))
	cfg.RequireEmailConfirmation = os.Getenv("REQUIRE_EMAIL_CONFIRMATION") == "true"
	cfg.SMTPHost = os.Getenv("SMTP_HOST")
	cfg.SMTPPort, _ = strconv.Atoi(os.Getenv("SMTP_PORT"))
	cfg.SMTPLogin = os.Getenv("SMTP_LOGIN")
	cfg.SMTPPassword = os.Getenv("SMTP_PASSWORD")

	Config = cfg
}
