package main

import (
	"log"

	"github.com/NikWaltz/metrics-collector/internal/api"
	"github.com/NikWaltz/metrics-collector/internal/service"
	"github.com/NikWaltz/metrics-collector/model"
)

func main() {
	myRepo := model.NewStorage()
	myService := service.New(myRepo)
	myAPI := api.New(myService)

	log.Fatalln(myAPI.Run())
}
