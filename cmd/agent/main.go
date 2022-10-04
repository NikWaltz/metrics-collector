package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"time"

	"github.com/caarlos0/env/v6"

	"github.com/NikWaltz/metrics-collector/model"
)

type Config struct {
	Address        string        `env:"ADDRESS"`
	PollInterval   time.Duration `env:"POLL_INTERVAL"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
}

var cfg Config

func init() {
	defaultPollInterval, _ := time.ParseDuration("2s")
	defaultReportInterval, _ := time.ParseDuration("10s")
	flag.StringVar(&cfg.Address, "a", "127.0.0.1:8080", "Server address for sending metrics")
	flag.DurationVar(&cfg.PollInterval, "p", defaultPollInterval, "Poll metrics interval")
	flag.DurationVar(&cfg.ReportInterval, "r", defaultReportInterval, "Sending report interval")
}

func main() {
	flag.Parse()
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("agent started")
	metricsCh := make(chan model.MetricsList)
	go scrapingTask(&cfg, metricsCh)
	go sendMetricsTask(&cfg, metricsCh)
	select {}
}

func scrapingTask(cfg *Config, ch chan model.MetricsList) {
	var metrics model.MetricsList
	ticker := time.NewTicker(cfg.PollInterval)
	for range ticker.C {
		log.Println("scraping metrics")
		ch <- scrape(&metrics)
	}
}

func sendMetricsTask(cfg *Config, ch chan model.MetricsList) {
	var endpoint string
	ticker := time.NewTicker(cfg.ReportInterval)
	metrics := <-ch
	for {
		select {
		case metrics = <-ch:
			log.Println("metrics updated")
		case <-ticker.C:
			v := reflect.ValueOf(metrics)
			for i := 0; i < v.NumField(); i++ {
				var metric model.Metrics
				switch v.Field(i).Kind() {
				case reflect.Float64:
					value := v.Field(i).Float()
					metric = model.Metrics{
						ID:    v.Type().Field(i).Name,
						MType: v.Field(i).Type().Name(),
						Delta: nil,
						Value: &value,
					}
				case reflect.Int64:
					value := v.Field(i).Int()
					metric = model.Metrics{
						ID:    v.Type().Field(i).Name,
						MType: v.Field(i).Type().Name(),
						Delta: &value,
						Value: nil,
					}
				default:
					log.Println("undefined metric type")
					continue
				}
				endpoint = fmt.Sprintf("http://%s/update/", cfg.Address)
				response := sendMetric(endpoint, &metric)
				if response == nil {
					continue
				}
				_, err := io.Copy(io.Discard, response.Body)
				if err != nil {
					log.Println(err)
				}
				response.Body.Close()
			}
		}
	}
}

func sendMetric(endpoint string, metrics *model.Metrics) *http.Response {
	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(*metrics)
	if err != nil {
		return nil
	}
	log.Println(body)
	response, err := http.Post(endpoint, "application/json", body)
	if err != nil {
		log.Println(err)
	}
	return response
}

func scrape(metrics *model.MetricsList) model.MetricsList {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
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
	return *metrics
}
