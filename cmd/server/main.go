package main

import (
	"github.com/NikWaltz/metrics-collector/internal/api"
	"github.com/NikWaltz/metrics-collector/internal/service"
	"github.com/NikWaltz/metrics-collector/internal/storage"
	"log"
)

func main() {
	myRepo := storage.New()
	myService := service.New(myRepo)
	myAPI := api.New(myService)

	log.Fatalln(myAPI.Run())
}
