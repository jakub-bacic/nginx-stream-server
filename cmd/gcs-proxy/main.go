package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/gcsblob"
)

var bufPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(nil)
	},
}

type uploadData struct {
	name string
	buf  *bytes.Buffer
}

func uploadToBlob(ctx context.Context, bucket *blob.Bucket, data uploadData) error {
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
		return err
	}

	n, err := io.Copy(obj, data.buf)
	if err != nil {
		return err
	}

	if err := obj.Close(); err != nil {
		return err
	}

	bufPool.Put(data.buf)

	fmt.Println(time.Now().UnixNano(), "UPLOAD END", data.name, "SIZE", n, "TOOK", time.Now().Sub(then).Milliseconds(), "ms")

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

		_, err := io.Copy(buf, r.Body)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}

		select {
		case <-r.Context().Done():
			writeError(w, http.StatusGatewayTimeout, err)
			return
		case c <- uploadData{name: name, buf: buf}:
			w.WriteHeader(http.StatusOK)
		}
	}
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

	http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), r)
}
