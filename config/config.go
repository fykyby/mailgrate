package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type config struct {
	AppName           string
	AppKey            string
	IsDev             bool
	Port              int
	DbUri             string
	WorkerCount       int
	JobTimeoutMinutes int
}

var Config *config

func InitConfig() {
	godotenv.Load()
	var cfg = new(config)

	cfg.AppName = os.Getenv("APP_NAME")
	cfg.AppKey = os.Getenv("APP_KEY")
	cfg.IsDev = os.Getenv("ENV") == "dev"
	cfg.Port, _ = strconv.Atoi(os.Getenv("PORT"))
	cfg.DbUri = os.Getenv("DB_URI")
	cfg.WorkerCount, _ = strconv.Atoi(os.Getenv("WORKER_COUNT"))
	cfg.JobTimeoutMinutes, _ = strconv.Atoi(os.Getenv("JOB_TIMEOUT_MINUTES"))

	Config = cfg
}
