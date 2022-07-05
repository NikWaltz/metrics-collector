package main

import (
	"github.com/stretchr/testify/assert"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_sendMetric(t *testing.T) {
	type args struct {
		endpoint   string
		wantStatus int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Send gauge metric",
			args: args{
				endpoint:   "/update/gauge/Alloc/54.23",
				wantStatus: 200,
			},
		},
		{
			name: "Send bad counter metric",
			args: args{
				endpoint:   "/update/counter/BadCount/54.23",
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
			response := sendMetric(server.URL + tt.args.endpoint)
			defer response.Body.Close()
			_, err := io.Copy(io.Discard, response.Body)
			if err != nil {
				log.Println(err)
			}
			assert.Equal(t, response.StatusCode, tt.args.wantStatus)
		})
	}
}
