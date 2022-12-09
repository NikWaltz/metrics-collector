package api

import (
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	Update(context.Context, string, string, string) error
	GetGauge(context.Context, string) (model.Gauge, error)
	GetCounter(context.Context, string) (model.Counter, error)
	GetStorage(context.Context) model.Storage
	Ping(ctx context.Context) error
	Close()
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
	key     string
}

func New(service Collector, key string) *api {
	r := chi.NewRouter()
	return &api{service: service, r: r, key: key}
}

func (a *api) updateHandle(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "value")
	err := a.service.Update(r.Context(), metricType, metricName, metricValue)
	if err != nil {
		var typeError *service.TypeError
		if errors.As(err, &typeError) {
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
	case model.GaugeType:
		if value, err := a.service.GetGauge(r.Context(), metricName); err == nil {
			w.WriteHeader(http.StatusOK)
			_, errWr := w.Write([]byte(fmt.Sprintf("%v", value)))
			if errWr != nil {
				log.Println(errWr)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	case model.CounterType:
		if value, err := a.service.GetCounter(r.Context(), metricName); err == nil {
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

	contentType, _ := header.ParseValueAndParams(r.Header, "Content-Type")
	if contentType != "application/json" {
		msg := "Content-Type header is not application/json"
		http.Error(w, msg, http.StatusUnsupportedMediaType)
		return
	}

	decodeErr := json.NewDecoder(r.Body).Decode(&metric)
	if decodeErr != nil {
		http.Error(w, decodeErr.Error(), http.StatusBadRequest)
		return
	}

	if a.key != "" {
		hashErr := verifyHash(&metric, a.key)
		if hashErr != nil {
			http.Error(w, hashErr.Error(), http.StatusBadRequest)
			return
		}
	}

	var value string
	switch strings.ToLower(metric.MType) {
	case model.GaugeType:
		value = strconv.FormatFloat(*metric.Value, 'f', -1, 64)
	case model.CounterType:
		value = strconv.FormatInt(*metric.Delta, 10)
	default:
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	err := a.service.Update(r.Context(), metric.MType, metric.ID, value)

	if err != nil {
		var typeError *service.TypeError
		if errors.As(err, &typeError) {
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

	contentType, _ := header.ParseValueAndParams(r.Header, "Content-Type")
	if contentType != "application/json" {
		msg := "Content-Type header is not application/json"
		http.Error(w, msg, http.StatusUnsupportedMediaType)
		return
	}

	decodeErr := json.NewDecoder(r.Body).Decode(&metric)
	if decodeErr != nil {
		http.Error(w, decodeErr.Error(), http.StatusBadRequest)
		return
	}

	switch strings.ToLower(metric.MType) {
	case model.GaugeType:
		if value, err := a.service.GetGauge(r.Context(), metric.ID); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			metric.Value = (*float64)(&value)
			if a.key != "" {
				hash(&metric, a.key)
			}
			errEncode := json.NewEncoder(w).Encode(metric)
			if errEncode != nil {
				log.Println(errEncode)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	case model.CounterType:
		if value, err := a.service.GetCounter(r.Context(), metric.ID); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			metric.Delta = (*int64)(&value)
			if a.key != "" {
				hash(&metric, a.key)
			}
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

func (a *api) updatesHandle(w http.ResponseWriter, r *http.Request) {
	var metrics []model.Metrics

	contentType, _ := header.ParseValueAndParams(r.Header, "Content-Type")
	if contentType != "application/json" {
		msg := "Content-Type header is not application/json"
		http.Error(w, msg, http.StatusUnsupportedMediaType)
		return
	}

	decodeErr := json.NewDecoder(r.Body).Decode(&metrics)
	if decodeErr != nil {
		http.Error(w, decodeErr.Error(), http.StatusBadRequest)
		return
	}

	for _, metric := range metrics {
		if a.key != "" {
			hashErr := verifyHash(&metric, a.key)
			if hashErr != nil {
				http.Error(w, hashErr.Error(), http.StatusBadRequest)
				return
			}
		}
		var value string
		switch strings.ToLower(metric.MType) {
		case model.GaugeType:
			value = strconv.FormatFloat(*metric.Value, 'f', -1, 64)
		case model.CounterType:
			value = strconv.FormatInt(*metric.Delta, 10)
		default:
			w.WriteHeader(http.StatusNotImplemented)
			return
		}
		err := a.service.Update(r.Context(), metric.MType, metric.ID, value)

		if err == nil {
			w.WriteHeader(http.StatusOK)
			return
		} else {
			var typeError *service.TypeError
			if errors.As(err, &typeError) {
				w.WriteHeader(http.StatusNotImplemented)
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}
			_, errWr := w.Write([]byte(err.Error()))
			if errWr != nil {
				log.Println(errWr)
			}
		}
	}
}

func (a *api) getMetricsHandle(w http.ResponseWriter, r *http.Request) {
	data := a.service.GetStorage(r.Context())
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

func (a *api) pingStoreHandle(w http.ResponseWriter, r *http.Request) {
	err := a.service.Ping(r.Context())
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func gzipCompressHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
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

func gzipDecompressHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, err := io.WriteString(w, err.Error())
			if err != nil {
				return
			}
			return
		}
		defer gz.Close()
		r.Body = gz
		next.ServeHTTP(w, r)
	})
}

func verifyHash(metric *model.Metrics, key string) error {
	var data []byte
	switch strings.ToLower(metric.MType) {
	case model.GaugeType:
		data = []byte(fmt.Sprintf("%s:gauge:%f", metric.ID, *metric.Value))
	case model.CounterType:
		data = []byte(fmt.Sprintf("%s:counter:%d", metric.ID, *metric.Delta))
	default:
		return errors.New("bad request")
	}
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	sign := h.Sum(nil)
	hash, err := hex.DecodeString(metric.Hash)
	if err != nil {
		return errors.New("bad request")
	}
	if hmac.Equal(sign, hash) {
		return nil
	} else {
		return errors.New("bad request")
	}
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

func (a *api) Run(addr string) error {
	a.r.Use(gzipCompressHandle)
	a.r.Use(gzipDecompressHandle)
	a.r.Post("/update/{type}/{name}/{value}", a.updateHandle)
	a.r.Get("/value/{type}/{name}", a.getValueHandle)
	a.r.Post("/update/", a.jsonUpdateHandle)
	a.r.Post("/updates/", a.updatesHandle)
	a.r.Post("/value/", a.getJSONValueHandle)
	a.r.Get("/", a.getMetricsHandle)
	a.r.Get("/ping", a.pingStoreHandle)
	return http.ListenAndServe(addr, a.r)
}
