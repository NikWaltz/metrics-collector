package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/NikWaltz/metrics-collector/model"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	Address        string        `env:"ADDRESS"`
	PollInterval   time.Duration `env:"POLL_INTERVAL"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
	Key            string        `env:"KEY"`
}

var cfg Config

func init() {
	const defaultPollInterval = time.Second * 2
	const defaultReportInterval = time.Second * 10
	flag.StringVar(&cfg.Address, "a", "127.0.0.1:8080", "Server address for sending metrics")
	flag.DurationVar(&cfg.PollInterval, "p", defaultPollInterval, "Poll metrics interval")
	flag.DurationVar(&cfg.ReportInterval, "r", defaultReportInterval, "Sending report interval")
	flag.StringVar(&cfg.Key, "k", "", "Key for hash")
}

func main() {
	flag.Parse()
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("agent started")
	metricsCh := make(chan model.MetricsList)
	extraMetricsCh := make(chan model.ExtraMetricsList)
	go scrapingTask(&cfg, metricsCh)
	go extraScrapingTask(&cfg, extraMetricsCh)
	go sendMetricsTask(&cfg, metricsCh, extraMetricsCh)
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
func extraScrapingTask(cfg *Config, ch chan model.ExtraMetricsList) {
	var metrics model.ExtraMetricsList
	ticker := time.NewTicker(cfg.PollInterval)
	for range ticker.C {
		log.Println("scraping metrics")
		ch <- extraScrape(&metrics)
	}
}

func sendMetricsTask(cfg *Config, ch chan model.MetricsList, ech chan model.ExtraMetricsList) {
	var endpoint string
	ticker := time.NewTicker(cfg.ReportInterval)
	metrics := <-ch
	extraMetrics := <-ech
	for {
		select {
		case metrics = <-ch:
			log.Println("metrics updated")
		case extraMetrics = <-ech:
			log.Println("metrics updated")
		case <-ticker.C:
			var metricsArray []*model.Metrics
			reflectMetrics := reflect.ValueOf(metrics)
			metricsArray = prepareMetricsArray(metricsArray, reflectMetrics, cfg.Key)
			reflectExtraMetrics := reflect.ValueOf(extraMetrics)
			metricsArray = prepareMetricsArray(metricsArray, reflectExtraMetrics, cfg.Key)
			endpoint = fmt.Sprintf("http://%s/updates/", cfg.Address)
			sendMetrics(endpoint, metricsArray)
		}
	}
}

func prepareMetricsArray(metricsArray []*model.Metrics, v reflect.Value, hashKey string) []*model.Metrics {
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

		if hashKey != "" {
			hash(&metric, cfg.Key)
		}
		metricsArray = append(metricsArray, &metric)
	}
	return metricsArray
}

func sendMetric(endpoint string, metrics *model.Metrics) *http.Response {
	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(metrics)
	if err != nil {
		log.Println(err)
		return nil
	}
	log.Println(body)
	response, err := http.Post(endpoint, "application/json", body)
	if err != nil {
		log.Println(err)
	}
	return response
}

func sendMetrics(endpoint string, metrics []*model.Metrics) {
	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(metrics)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(body)
	response, err := http.Post(endpoint, "application/json", body)
	if err != nil {
		log.Println(err)
	}
	if response == nil {
		log.Println("Response is nil")
		return
	}
	defer response.Body.Close()
	_, errDiscard := io.Copy(io.Discard, response.Body)
	if errDiscard != nil {
		log.Println(err)
	}
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

func extraScrape(metrics *model.ExtraMetricsList) model.ExtraMetricsList {
	v, _ := mem.VirtualMemory()
	s, _ := cpu.Percent(0, false)
	metrics.TotalMemory = model.Gauge(v.Total)
	metrics.FreeMemory = model.Gauge(v.Free)
	metrics.CPUutilization1 = model.Gauge(s[0])
	return *metrics
}

func hash(metric *model.Metrics, key string) {
	var data []byte
	switch strings.ToLower(metric.MType) {
	case model.GaugeType:
		data = []byte(fmt.Sprintf("%s:gauge:%f", metric.ID, *metric.Value))
	case model.CounterType:
		data = []byte(fmt.Sprintf("%s:counter:%d", metric.ID, *metric.Delta))
	default:
		return
	}
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	metric.Hash = hex.EncodeToString(h.Sum(nil))
}
