package main

import (
	"github.com/NikWaltz/metrics-collector/model"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_sendMetric(t *testing.T) {
	value := 54.23
	type args struct {
		endpoint   string
		metrics    model.Metrics
		wantStatus int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Send gauge metric",
			args: args{
				endpoint: "/update/",
				metrics: model.Metrics{
					ID:    "Alloc",
					MType: "Gauge",
					Delta: nil,
					Value: &value,
				},
				wantStatus: 200,
			},
		},
		{
			name: "Send bad counter metric",
			args: args{
				endpoint: "/update/",
				metrics: model.Metrics{
					ID:    "Alloc",
					MType: "Counter",
					Delta: nil,
					Value: &value,
				},
				wantStatus: 400,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				assert.Equal(t, req.URL.String(), tt.args.endpoint)
				rw.WriteHeader(tt.args.wantStatus)
			}))
			defer server.Close()
			response := sendMetric(server.URL+tt.args.endpoint, &tt.args.metrics)
			defer response.Body.Close()
			_, err := io.Copy(io.Discard, response.Body)
			if err != nil {
				log.Println(err)
			}
			assert.Equal(t, response.StatusCode, tt.args.wantStatus)
		})
	}
}
