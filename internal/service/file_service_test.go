package service

import (
	"encoding/json"
	"fmt"
	"github.com/NikWaltz/metrics-collector/model"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func Test_fileService_saveToFile(t *testing.T) {
	d, _ := time.ParseDuration("5s")

	type fields struct {
		storage       *model.Storage
		fileName      string
		storeInterval time.Duration
		restore       bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "Update gauge metric",
			fields: fields{
				storage:       &model.Storage{},
				fileName:      "tmp.json",
				storeInterval: d,
				restore:       false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewFileService(tt.fields.storage, tt.fields.fileName, tt.fields.storeInterval, tt.fields.restore)
			file, err := os.OpenFile(p.fileName, os.O_RDWR|os.O_CREATE, 0777)
			if err != nil {
				fmt.Println(err)
			}
			storage := &model.Storage{
				Gauges:   map[string]model.Gauge{"Alloc": 53.23, "Mem": 45.2},
				Counters: map[string]model.Counter{"PollCount": 10},
			}
			errEncode := json.NewEncoder(file).Encode(storage)
			if errEncode != nil {
				fmt.Println(err)
			}
			p.readFromFile()
			defer file.Close()
			defer os.Remove(tt.fields.fileName)
			alloc, _ := tt.fields.storage.GetGauge("Alloc")
			mem, _ := tt.fields.storage.GetGauge("Mem")
			pollCount, _ := tt.fields.storage.GetCounter("PollCount")
			assert.Equalf(t, model.Gauge(53.23), alloc, "Wrong Alloc value")
			assert.Equalf(t, model.Gauge(45.2), mem, "Wrong Mem value")
			assert.Equalf(t, model.Counter(10), pollCount, "Wrong PollCount value")
		})
	}
}

func Test_fileService_readFromFile(t *testing.T) {
	interval, _ := time.ParseDuration("5s")

	type fields struct {
		storage       *model.Storage
		fileName      string
		storeInterval time.Duration
		restore       bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "Update gauge metric",
			fields: fields{
				storage: &model.Storage{
					Gauges:   map[string]model.Gauge{"Alloc": 53.23, "Mem": 45.2},
					Counters: map[string]model.Counter{"PollCount": 10},
				},
				fileName:      "tmp.json",
				storeInterval: interval,
				restore:       false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewFileService(tt.fields.storage, tt.fields.fileName, tt.fields.storeInterval, tt.fields.restore)
			p.saveToFile()
			file, err := os.OpenFile(p.fileName, os.O_RDWR, 0777)
			if err != nil {
				fmt.Println(err)
			}
			defer file.Close()
			defer os.Remove(tt.fields.fileName)
			storage := &model.Storage{}
			errDecode := json.NewDecoder(file).Decode(storage)
			if errDecode != nil {
				fmt.Println(err)
			}
			alloc, _ := storage.GetGauge("Alloc")
			mem, _ := storage.GetGauge("Mem")
			pollCount, _ := storage.GetCounter("PollCount")
			assert.Equalf(t, model.Gauge(53.23), alloc, "Wrong Alloc value")
			assert.Equalf(t, model.Gauge(45.2), mem, "Wrong Mem value")
			assert.Equalf(t, model.Counter(10), pollCount, "Wrong PollCount value")
		})
	}
}
