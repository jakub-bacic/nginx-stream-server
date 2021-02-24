# nginx-stream-server

A very simple live-streaming server based on nginx.

## Run it locally using Docker

```
docker run --rm -it -p 8080:80 -p 1935:1935 -e PUBLISH_PASSWORD=my-secret jbacic/nginx-stream-server
```

**Server**: `rtmp://localhost/live`

**Stream Key**: `<your_key>?pwd=my-secret`

where `<your-key>` is an unique stream name.

HLS stream can be accessed at:

`http://localhost:8080/hls/<your-key>.m3u8`

## Run it on Kubernetes (helm chart)

```
git clone git@github.com:jakub-bacic/nginx-stream-server.git
cd nginx-stream-server
helm install --set publishPassword=<my-secret> my-release ./charts/nginx-stream-server
```

## How to start streaming?

1. Install and launch [OBS Studio](https://obsproject.com/)
2. Click on `Settings`.
3. Enter `Stream` tab.
4. Choose `Custom...` as `Service` and fill `Server` and `Stream Key` accordingly.
5. Save the configuration and click `Start Streaming`.
