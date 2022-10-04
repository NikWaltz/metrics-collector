package api

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/golang/gddo/httputil/header"

	"github.com/NikWaltz/metrics-collector/internal/service"
	"github.com/NikWaltz/metrics-collector/model"
)

type Collector interface {
	Update(string, string, string) error
	GetGauge(string) (model.Gauge, error)
	GetCounter(string) (model.Counter, error)
	GetStorage() model.Storage
}

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

type api struct {
	r       chi.Router
	service Collector
}

func New(service Collector) *api {
	r := chi.NewRouter()

	return &api{service: service, r: r}
}

func (a *api) updateHandle(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "value")
	err := a.service.Update(metricType, metricName, metricValue)
	if err != nil {
		if _, ok := err.(*service.TypeError); ok {
			w.WriteHeader(http.StatusNotImplemented)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
		_, err := w.Write([]byte(err.Error()))
		if err != nil {
			log.Println(err)
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func (a *api) getValueHandle(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	switch strings.ToLower(metricType) {
	case "gauge":
		if value, err := a.service.GetGauge(metricName); err == nil {
			w.WriteHeader(http.StatusOK)
			_, errWr := w.Write([]byte(fmt.Sprintf("%v", value)))
			if errWr != nil {
				log.Println(errWr)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	case "counter":
		if value, err := a.service.GetCounter(metricName); err == nil {
			w.WriteHeader(http.StatusOK)
			_, errWr := w.Write([]byte(fmt.Sprintf("%d", value)))
			if errWr != nil {
				log.Println(errWr)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (a *api) jsonUpdateHandle(w http.ResponseWriter, r *http.Request) {
	var metric model.Metrics

	if r.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
		if value != "application/json" {
			msg := "Content-Type header is not application/json"
			http.Error(w, msg, http.StatusUnsupportedMediaType)
			return
		}
	}

	decodeErr := json.NewDecoder(r.Body).Decode(&metric)
	if decodeErr != nil {
		http.Error(w, decodeErr.Error(), http.StatusBadRequest)
		return
	}

	var value string
	switch strings.ToLower(metric.MType) {
	case "gauge":
		value = strconv.FormatFloat(*metric.Value, 'f', -1, 64)
	case "counter":
		value = strconv.FormatInt(*metric.Delta, 10)
	default:
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	err := a.service.Update(metric.MType, metric.ID, value)
	if err != nil {
		if _, ok := err.(*service.TypeError); ok {
			w.WriteHeader(http.StatusNotImplemented)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
		_, err := w.Write([]byte(err.Error()))
		if err != nil {
			log.Println(err)
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func (a *api) getJSONValueHandle(w http.ResponseWriter, r *http.Request) {
	var metric model.Metrics

	if r.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
		if value != "application/json" {
			msg := "Content-Type header is not application/json"
			http.Error(w, msg, http.StatusUnsupportedMediaType)
			return
		}
	}

	decodeErr := json.NewDecoder(r.Body).Decode(&metric)
	if decodeErr != nil {
		http.Error(w, decodeErr.Error(), http.StatusBadRequest)
		return
	}

	switch strings.ToLower(metric.MType) {
	case "gauge":
		if value, err := a.service.GetGauge(metric.ID); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			metric.Value = (*float64)(&value)
			errEncode := json.NewEncoder(w).Encode(metric)
			if errEncode != nil {
				log.Println(errEncode)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	case "counter":
		if value, err := a.service.GetCounter(metric.ID); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			metric.Delta = (*int64)(&value)
			errEncode := json.NewEncoder(w).Encode(metric)
			if errEncode != nil {
				log.Println(errEncode)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (a *api) getMetricsHandle(w http.ResponseWriter, r *http.Request) {
	data := a.service.GetStorage()
	data.Counters["PollCount"] = 24
	htmlTemplate := `{{range $index, $element := .Gauges}}{{$index}} {{printf "%f" $element}}
{{end}}{{range $index, $element := .Counters}}{{$index}} {{printf "%d" $element}}
{{end}}`
	tmpl, err := template.New("metrics").Parse(htmlTemplate)
	if err != nil {
		log.Println(err)
	}
	w.Header().Set("Content-Type", "text/html")
	errExec := tmpl.Execute(w, &data)
	if errExec != nil {
		log.Println(err)
	}
}

func gzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			_, err := io.WriteString(w, err.Error())
			if err != nil {
				return
			}
			return
		}
		defer gz.Close()

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

func (a *api) Run(addr string) error {
	a.r.Use(gzipHandle)
	a.r.Post("/update/{type}/{name}/{value}", a.updateHandle)
	a.r.Get("/value/{type}/{name}", a.getValueHandle)
	a.r.Post("/update/", a.jsonUpdateHandle)
	a.r.Post("/value/", a.getJSONValueHandle)
	a.r.Get("/", a.getMetricsHandle)
	return http.ListenAndServe(addr, a.r)
}
