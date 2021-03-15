package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/jakub-bacic/nginx-stream-server/pkg/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/gcsblob"
)

var (
	storageBytes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "openmind_storage_total_bytes",
			Help: "The total bytes transfered to the storage",
		},
		[]string{"id"},
	)

	transferCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "openmind_storage_transfer_total",
			Help: "The total count of transfers to the storage",
		},
		[]string{"id", "status"},
	)

	transferInflight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openmind_storage_transfer_inflight",
			Help: "The total count of inflight transfers to the storage",
		},
		[]string{"id"},
	)

	transferLatencies = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "openmind_storage_transfer_latency",
			Help:       "Transfer latencies",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"id"},
	)

	transcoderRealtimeCoeff = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "openmind_transcoder_realtime_coefficient",
			Help:       "Transcoder realtime coefficient",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"id"},
	)
)

var bufPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(nil)
	},
}

type uploadData struct {
	id   string
	name string
	buf  *bytes.Buffer
}

func uploadToBlob(ctx context.Context, bucket *blob.Bucket, data uploadData) error {
	transferInflight.WithLabelValues(data.id).Inc()
	defer transferInflight.WithLabelValues(data.id).Dec()

	then := time.Now()

	var contentType string
	switch filepath.Ext(data.name) {
	case ".m3u8":
		contentType = "application/x-mpegURL"
	case ".ts":
		contentType = "video/MP2T"
	default:
		contentType = "application/octet-stream"
	}

	obj, err := bucket.NewWriter(
		ctx,
		data.name,
		&blob.WriterOptions{
			BufferSize:   1024 * 1024 * 8,
			CacheControl: "no-store",
			ContentType:  contentType,
		},
	)
	if err != nil {
		transferCount.WithLabelValues(data.id, "failure").Inc()
		return err
	}

	n, err := io.Copy(obj, data.buf)
	if err != nil {
		transferCount.WithLabelValues(data.id, "failure").Inc()
		return err
	}

	if err := obj.Close(); err != nil {
		transferCount.WithLabelValues(data.id, "failure").Inc()
		return err
	}

	bufPool.Put(data.buf)

	lat := time.Now().Sub(then)

	transferLatencies.WithLabelValues(data.id).Observe(lat.Seconds())
	storageBytes.WithLabelValues(data.id).Add(float64(n))
	transferCount.WithLabelValues(data.id, "success").Inc()

	fmt.Println(time.Now().UnixNano(), "UPLOAD END", data.name, "SIZE", n, "TOOK", lat.Milliseconds(), "ms")

	return nil
}

func writeError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	fmt.Fprintf(w, "%d: %s", status, err)
	fmt.Println(time.Now().UnixNano(), "Error:", err)
}

func putUpload(c chan<- uploadData) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		name := path.Join(chi.URLParam(r, "name"), chi.URLParam(r, "file"))

		buf := bufPool.Get().(*bytes.Buffer)
		buf.Reset()

		_, err := io.Copy(buf, r.Body)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}

		select {
		case <-r.Context().Done():
			writeError(w, http.StatusGatewayTimeout, err)
			return
		case c <- uploadData{id: chi.URLParam(r, "name"), name: name, buf: buf}:
			w.WriteHeader(http.StatusOK)
		}
	}
}

func postTranscoderStats(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var stats model.TranscoderStats
	if err := json.NewDecoder(r.Body).Decode(&stats); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	transcoderRealtimeCoeff.WithLabelValues(stats.Id).Observe(stats.RealtimeCoefficient)

	fmt.Println(time.Now().UnixNano(), "ID", stats.Id, "COEFF", stats.RealtimeCoefficient)

	w.WriteHeader(http.StatusAccepted)
}

func main() {
	bucket, err := blob.OpenBucket(context.Background(), os.Getenv("GCS_BUCKET"))
	if err != nil {
		panic(err)
	}
	defer bucket.Close()

	c := make(chan uploadData, 16)
	for i := 0; i < 16; i++ {
		go func() {
			for {
				data := <-c
				ctx, cf := context.WithTimeout(context.Background(), time.Second*10)
				if err := uploadToBlob(ctx, bucket, data); err != nil {
					fmt.Println("Error:", err)
				}
				cf()
			}
		}()
	}

	r := chi.NewRouter()
	r.Put("/upload/{name}/{file}", putUpload(c))
	r.Post("/transcoder/stats", postTranscoderStats)
	r.Handle("/metrics", promhttp.Handler())

	http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), r)
}
