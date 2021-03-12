package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/go-chi/chi"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/gcsblob"
)

func writeError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	fmt.Fprintf(w, "%d: %s", status, err)
	fmt.Println(time.Now().UnixNano(), "Error:", err)
}

func putUpload(bucket *blob.Bucket) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		then := time.Now()

		p := path.Join(chi.URLParam(r, "name"), chi.URLParam(r, "file"))

		var contentType string
		switch filepath.Ext(chi.URLParam(r, "file")) {
		case ".m3u8":
			contentType = "application/x-mpegURL"
		case ".ts":
			contentType = "video/MP2T"
		default:
			contentType = "application/octet-stream"
		}

		obj, err := bucket.NewWriter(
			context.Background(),
			p,
			&blob.WriterOptions{
				BufferSize:   1024 * 1024 * 8,
				CacheControl: "no-store",
				ContentType:  contentType,
			},
		)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}

		fmt.Println(time.Now().UnixNano(), "UPLOAD START", p)

		n, err := io.Copy(obj, r.Body)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}

		if err := obj.Close(); err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}

		fmt.Println(time.Now().UnixNano(), "UPLOAD END", p, "SIZE", n, "TOOK", time.Now().Sub(then).Milliseconds(), "ms")

		// This is crucial.
		w.WriteHeader(http.StatusOK)
	}
}

func main() {
	bucket, err := blob.OpenBucket(context.Background(), os.Getenv("GCS_BUCKET"))
	if err != nil {
		panic(err)
	}
	defer bucket.Close()

	r := chi.NewRouter()
	r.Put("/upload/{name}/{file}", putUpload(bucket))

	http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), r)
}
