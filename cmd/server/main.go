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
	DatabaseDsn   string        `env:"DATABASE_DSN"`
	Key           string        `env:"KEY"`
}

var cfg Config

func init() {
	const defaultDuration = time.Second * 300
	flag.StringVar(&cfg.Address, "a", "127.0.0.1:8080", "Server address")
	flag.DurationVar(&cfg.StoreInterval, "i", defaultDuration, "Store to file interval")
	flag.StringVar(&cfg.StoreFile, "f", "/tmp/devops-metrics-db.json", "Store file path")
	flag.BoolVar(&cfg.Restore, "r", true, "Restore storage from file")
	flag.StringVar(&cfg.DatabaseDsn, "d", "", "Data source name")
	flag.StringVar(&cfg.Key, "k", "", "Key for hash")
	flag.Parse()
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	log.Println("server started")

	myRepo := model.NewStorage()
	var myService api.Collector

	if cfg.DatabaseDsn != "" {
		myService = service.NewDBService(myRepo, cfg.DatabaseDsn)
	} else {
		myService = service.NewService(myRepo)
		myFileService := service.NewFileService(myRepo, cfg.StoreFile, cfg.StoreInterval, cfg.Restore)
		go myFileService.Run()
	}

	myAPI := api.New(myService, cfg.Key)
	err := myAPI.Run(cfg.Address)
	if err != nil {
		log.Fatalln(err)
	}
}
