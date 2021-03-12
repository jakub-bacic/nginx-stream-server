package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
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

func putUpload(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	then := time.Now()

	bucket, err := blob.OpenBucket(context.Background(), os.Getenv("GCS_BUCKET"))
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	defer bucket.Close()

	p := path.Join(chi.URLParam(r, "name"), chi.URLParam(r, "file"))

	obj, err := bucket.NewWriter(
		r.Context(),
		p,
		nil,
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

	w.WriteHeader(http.StatusOK)

	fmt.Println(time.Now().UnixNano(), "UPLOAD END", p, "SIZE", n, "TOOK", time.Now().Sub(then).Milliseconds(), "ms")
}

func main() {
	r := chi.NewRouter()
	r.Put("/upload/{name}/{file}", putUpload)

	http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), r)
}
