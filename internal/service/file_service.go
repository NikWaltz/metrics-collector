package service

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/NikWaltz/metrics-collector/model"
)

type fileService struct {
	storage       *model.Storage
	fileName      string
	storeInterval time.Duration
	restore       bool
}

func NewFileService(storage *model.Storage, fileName string, storeInterval time.Duration, restore bool) *fileService {
	fileService := &fileService{
		storage:       storage,
		fileName:      fileName,
		storeInterval: storeInterval,
		restore:       restore,
	}
	if restore {
		fileService.readFromFile()
	}
	return fileService
}

func (p *fileService) saveToFile() {
	file, err := os.OpenFile(p.fileName, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	errEncode := json.NewEncoder(file).Encode(p.storage)
	if errEncode != nil {
		log.Println(err)
	}
}

func (p *fileService) readFromFile() {
	file, err := os.OpenFile(p.fileName, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	errDecode := json.NewDecoder(file).Decode(p.storage)
	if errDecode != nil {
		log.Println(err)
	}
}

func (p *fileService) Run() {
	ticker := time.NewTicker(p.storeInterval)
	for range ticker.C {
		p.saveToFile()
	}
}
