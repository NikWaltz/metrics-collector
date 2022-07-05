package main

import (
	"bytes"
	"fmt"
	"github.com/NikWaltz/metrics-collector/model"
	"io"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"time"
)

var metrics model.Metrics

type Config struct {
	host           string
	port           int
	pollInterval   int
	reportInterval int
}

func main() {
	var stats runtime.MemStats
	cfg := Config{
		host:           "localhost",
		port:           8080,
		pollInterval:   2,
		reportInterval: 10,
	}
	go sendMetricsTask(&cfg)
	for {
		log.Println("Scraping metrics")
		scrape(&metrics, &stats)
		time.Sleep(time.Duration(cfg.pollInterval) * time.Second)
	}
}

func sendMetricsTask(cfg *Config) {
	ticker := time.NewTicker(time.Duration(cfg.reportInterval) * time.Second)
	for range ticker.C {
		var endpoint string
		v := reflect.ValueOf(metrics)
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).Kind() == reflect.Float64 {
				endpoint = fmt.Sprintf("http://%s:%d/update/%s/%s/%f", cfg.host, cfg.port, v.Field(i).Type().Name(), v.Type().Field(i).Name, v.Field(i).Float())
			} else {
				endpoint = fmt.Sprintf("http://%s:%d/update/%s/%s/%d", cfg.host, cfg.port, v.Field(i).Type().Name(), v.Type().Field(i).Name, v.Field(i).Int())
			}
			log.Printf("Sending to %s\n", endpoint)
			response := sendMetric(endpoint)
			_, err := io.Copy(io.Discard, response.Body)
			if err != nil {
				log.Println(err)
			}
			response.Body.Close()
		}
	}
}

func sendMetric(endpoint string) *http.Response {
	response, err := http.Post(endpoint, "text/plain", bytes.NewBufferString(""))
	if err != nil {
		log.Println(err)
	}
	return response
}

func scrape(metrics *model.Metrics, stats *runtime.MemStats) {
	runtime.ReadMemStats(stats)
	metrics.Alloc = model.Gauge(stats.Alloc)
	metrics.BuckHashSys = model.Gauge(stats.BuckHashSys)
	metrics.Frees = model.Gauge(stats.Frees)
	metrics.GCCPUFraction = model.Gauge(stats.GCCPUFraction)
	metrics.GCSys = model.Gauge(stats.GCSys)
	metrics.HeapAlloc = model.Gauge(stats.HeapAlloc)
	metrics.HeapIdle = model.Gauge(stats.HeapIdle)
	metrics.HeapInuse = model.Gauge(stats.HeapInuse)
	metrics.HeapObjects = model.Gauge(stats.HeapObjects)
	metrics.HeapReleased = model.Gauge(stats.HeapReleased)
	metrics.HeapSys = model.Gauge(stats.HeapSys)
	metrics.LastGC = model.Gauge(stats.LastGC)
	metrics.Lookups = model.Gauge(stats.Lookups)
	metrics.MCacheInuse = model.Gauge(stats.MCacheInuse)
	metrics.MCacheSys = model.Gauge(stats.MCacheSys)
	metrics.MSpanInuse = model.Gauge(stats.MSpanInuse)
	metrics.MSpanSys = model.Gauge(stats.MSpanSys)
	metrics.Mallocs = model.Gauge(stats.Mallocs)
	metrics.NextGC = model.Gauge(stats.NextGC)
	metrics.NumForcedGC = model.Gauge(stats.NumForcedGC)
	metrics.NumGC = model.Gauge(stats.NumGC)
	metrics.OtherSys = model.Gauge(stats.OtherSys)
	metrics.PauseTotalNs = model.Gauge(stats.PauseTotalNs)
	metrics.StackInuse = model.Gauge(stats.StackInuse)
	metrics.StackSys = model.Gauge(stats.StackSys)
	metrics.Sys = model.Gauge(stats.Sys)
	metrics.TotalAlloc = model.Gauge(stats.TotalAlloc)
	metrics.PollCount++
	metrics.RandomValue = model.Gauge(rand.Float64())
}
