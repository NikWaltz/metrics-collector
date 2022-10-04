package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/NikWaltz/metrics-collector/model"
)

type mockCollector struct {
	err error
	st  model.Storage
}

func (c mockCollector) Update(name string, typ string, value string) error {
	return c.err
}

func (c mockCollector) GetGauge(name string) (model.Gauge, error) {
	return c.st.Gauges[name], c.err
}
func (c mockCollector) GetCounter(name string) (model.Counter, error) {
	return c.st.Counters[name], c.err
}
func (c mockCollector) GetStorage() model.Storage {
	return c.st
}

func Test_updateHandle(t *testing.T) {
	type fields struct {
		service Collector
	}
	tests := []struct {
		name           string
		fields         fields
		wantStatusCode int
	}{
		{
			name:           "Handle without errors",
			fields:         fields{mockCollector{err: nil}},
			wantStatusCode: 200,
		},
		{
			name:           "Handle with errors",
			fields:         fields{mockCollector{err: errors.New("wrong types")}},
			wantStatusCode: 400,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := New(tt.fields.service)
			assert.HTTPStatusCode(t, a.updateHandle, http.MethodPost, "/update/type/name/value", nil, tt.wantStatusCode)
		})
	}
}

func Test_getValueHandle(t *testing.T) {
	stor := model.Storage{
		Gauges:   map[string]model.Gauge{"Alloc": 43.53234, "Mem": 72},
		Counters: map[string]model.Counter{"PollCounter": 5},
	}
	type fields struct {
		r       chi.Router
		service Collector
	}
	tests := []struct {
		name           string
		fields         fields
		metricType     string
		metricName     string
		wantStatusCode int
		wantBody       string
	}{
		{
			name: "Get counter value",
			fields: fields{
				r: chi.NewRouter(),
				service: mockCollector{
					err: nil,
					st:  stor,
				},
			},
			metricType:     "counter",
			metricName:     "PollCounter",
			wantStatusCode: 200,
			wantBody:       "5",
		},
		{
			name: "Get gauge value",
			fields: fields{
				r: chi.NewRouter(),
				service: mockCollector{
					err: nil,
					st:  stor,
				},
			},
			metricType:     "gauge",
			metricName:     "Alloc",
			wantStatusCode: 200,
			wantBody:       "43.53234",
		},
		{
			name: "Get wrong type",
			fields: fields{
				r: chi.NewRouter(),
				service: mockCollector{
					err: nil,
					st:  stor,
				},
			},
			metricType:     "histogram",
			metricName:     "Alloc",
			wantStatusCode: 404,
			wantBody:       "",
		},
		{
			name: "Get wrong metric",
			fields: fields{
				r: chi.NewRouter(),
				service: mockCollector{
					err: errors.New("some error"),
					st:  stor,
				},
			},
			metricType:     "gauge",
			metricName:     "Hello",
			wantStatusCode: 404,
			wantBody:       "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &api{
				r:       tt.fields.r,
				service: tt.fields.service,
			}
			a.r.Get("/value/{type}/{name}", a.getValueHandle)

			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/value/%s/%s", tt.metricType, tt.metricName), nil)
			if err != nil {
				t.Fatal(err)
			}
			rr := httptest.NewRecorder()
			a.r.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatusCode, rr.Code)
			assert.Equal(t, tt.wantBody, rr.Body.String())
		})
	}
}

func Test_jsonUpdateHandle(t *testing.T) {
	floatValue := 43.53234
	intValue := int64(55)
	type fields struct {
		r       chi.Router
		service Collector
	}
	tests := []struct {
		name           string
		metrics        model.Metrics
		fields         fields
		wantStatusCode int
	}{
		{
			name: "Update gauge",
			metrics: model.Metrics{
				ID:    "Alloc",
				MType: "Gauge",
				Value: &floatValue,
			},
			fields: fields{
				r: chi.NewRouter(),
				service: mockCollector{
					err: nil,
				},
			},
			wantStatusCode: 200,
		},
		{
			name: "Update counter",
			metrics: model.Metrics{
				ID:    "PollCounter",
				MType: "Counter",
				Delta: &intValue,
			},
			fields: fields{
				r: chi.NewRouter(),
				service: mockCollector{
					err: nil,
				},
			},
			wantStatusCode: 200,
		},
		{
			name: "Update wrong type",
			metrics: model.Metrics{
				ID:    "Metric",
				MType: "Histogram",
				Value: &floatValue,
			},
			fields: fields{
				r: chi.NewRouter(),
				service: mockCollector{
					err: nil,
				},
			},
			wantStatusCode: 501,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &api{
				r:       tt.fields.r,
				service: tt.fields.service,
			}
			a.r.Post("/update/", a.jsonUpdateHandle)
			body := new(bytes.Buffer)
			err := json.NewEncoder(body).Encode(tt.metrics)
			if err != nil {
				return
			}
			req, err := http.NewRequest(http.MethodPost, "/update/", body)
			if err != nil {
				t.Fatal(err)
			}
			rr := httptest.NewRecorder()
			a.r.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatusCode, rr.Code)
			//assert.HTTPStatusCode(t, a.jsonUpdateHandle, http.MethodPost, "/update/", body, tt.wantStatusCode)
		})
	}
}

func Test_getJsonValueHandle(t *testing.T) {
	stor := model.Storage{
		Gauges:   map[string]model.Gauge{"Alloc": 43.53234, "Mem": 72},
		Counters: map[string]model.Counter{"PollCounter": 5},
	}
	type fields struct {
		r       chi.Router
		service Collector
	}
	tests := []struct {
		name           string
		fields         fields
		metrics        model.Metrics
		wantStatusCode int
		wantBody       string
	}{
		{
			name: "Get counter value",
			fields: fields{
				r: chi.NewRouter(),
				service: mockCollector{
					err: nil,
					st:  stor,
				},
			},
			metrics: model.Metrics{
				ID:    "PollCounter",
				MType: "Counter",
			},
			wantStatusCode: 200,
			wantBody:       "{\"id\":\"PollCounter\",\"type\":\"Counter\",\"delta\":5}\n",
		},
		{
			name: "Get gauge value",
			fields: fields{
				r: chi.NewRouter(),
				service: mockCollector{
					err: nil,
					st:  stor,
				},
			},
			metrics: model.Metrics{
				ID:    "Alloc",
				MType: "Gauge",
			},
			wantStatusCode: 200,
			wantBody:       "{\"id\":\"Alloc\",\"type\":\"Gauge\",\"value\":43.53234}\n",
		},
		{
			name: "Get wrong type",
			fields: fields{
				r: chi.NewRouter(),
				service: mockCollector{
					err: nil,
					st:  stor,
				},
			},
			metrics: model.Metrics{
				ID:    "Alloc",
				MType: "Histogram",
			},
			wantStatusCode: 404,
			wantBody:       "",
		},
		{
			name: "Get wrong metric",
			fields: fields{
				r: chi.NewRouter(),
				service: mockCollector{
					err: errors.New("some error"),
					st:  stor,
				},
			},
			metrics: model.Metrics{
				ID:    "Hello",
				MType: "Gauge",
			},
			wantStatusCode: 404,
			wantBody:       "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &api{
				r:       tt.fields.r,
				service: tt.fields.service,
			}
			a.r.Post("/value/", a.getJSONValueHandle)

			body := new(bytes.Buffer)
			err := json.NewEncoder(body).Encode(tt.metrics)
			if err != nil {
				return
			}
			req, err := http.NewRequest(http.MethodPost, "/value/", body)
			if err != nil {
				t.Fatal(err)
			}
			rr := httptest.NewRecorder()
			a.r.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatusCode, rr.Code)
			assert.Equal(t, tt.wantBody, rr.Body.String())
		})
	}
}

func Test_getMetricsHandle(t *testing.T) {
	stor := model.Storage{
		Gauges:   map[string]model.Gauge{"Alloc": 43.53234, "Mem": 72},
		Counters: map[string]model.Counter{"PollCounter": 5},
	}
	type fields struct {
		r       chi.Router
		service Collector
	}
	tests := []struct {
		name           string
		fields         fields
		metricType     string
		metricName     string
		wantStatusCode int
		wantBody       string
	}{
		{
			name: "Get metrics",
			fields: fields{
				r: chi.NewRouter(),
				service: mockCollector{
					err: nil,
					st:  stor,
				},
			},
			wantStatusCode: 200,
			wantBody:       "Alloc 43.532340\nMem 72.000000\nPollCount 24\nPollCounter 5\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &api{
				r:       tt.fields.r,
				service: tt.fields.service,
			}
			a.r.Get("/", a.getMetricsHandle)

			req, err := http.NewRequest(http.MethodGet, "/", nil)
			if err != nil {
				t.Fatal(err)
			}
			rr := httptest.NewRecorder()
			a.r.ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatusCode, rr.Code)
			assert.Equal(t, tt.wantBody, rr.Body.String())
		})
	}
}
