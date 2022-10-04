package main

import (
	"flag"
	"log"
	"time"

	"github.com/caarlos0/env/v6"

	"github.com/NikWaltz/metrics-collector/internal/api"
	"github.com/NikWaltz/metrics-collector/internal/service"
	"github.com/NikWaltz/metrics-collector/model"
)

type Config struct {
	Address       string        `env:"ADDRESS"`
	StoreInterval time.Duration `env:"STORE_INTERVAL"`
	StoreFile     string        `env:"STORE_FILE"`
	Restore       bool          `env:"RESTORE"`
}

var cfg Config

func init() {
	defaultDuration, _ := time.ParseDuration("300s")
	flag.StringVar(&cfg.Address, "a", "127.0.0.1:8080", "Server address")
	flag.DurationVar(&cfg.StoreInterval, "i", defaultDuration, "Store to file interval")
	flag.StringVar(&cfg.StoreFile, "f", "/tmp/devops-metrics-db.json", "Store file path")
	flag.BoolVar(&cfg.Restore, "r", true, "Restore storage from file")
	flag.Parse()
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	log.Println("server started")

	myRepo := model.NewStorage()
	myFileService := service.NewFileService(myRepo, cfg.StoreFile, cfg.StoreInterval, cfg.Restore)
	myService := service.NewService(myRepo)
	myAPI := api.New(myService)

	go myFileService.Run()
	log.Fatalln(myAPI.Run(cfg.Address))
}
