package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/jakub-bacic/nginx-stream-server/pkg/model"
)

var (
	speedRe = regexp.MustCompile(`speed=\d+\.\d+x`)
)

func parseSpeed(line string) (ok bool, speed float64) {
	s := speedRe.FindString(line)
	f, err := strconv.ParseFloat(strings.TrimRight(strings.TrimLeft(s, "speed="), "x"), 64)
	if err != nil {
		return
	}
	return true, f
}

func insert(a []string, s string, i int) []string {
	return append(a[:i], append([]string{s}, a[i:]...)...)
}

func run() int {
	if len(os.Args) != 5 {
		fmt.Println("Usage:", os.Args[0], "<streamId>", "<proxyAddress>", "<segmentDuration>", "<segmentCount>")
		return 2
	}

	var args = `-re -i rtmp://localhost:1935/live/$1
    -fflags nobuffer -flags low_delay -strict experimental -probesize 32 -analyzeduration 0
    -max_muxing_queue_size 400
    -map 0:v:0 -map 0:a:0 -map 0:v:0 -map 0:a:0 -map 0:v:0 -map 0:a:0
    -c:v libx264 -tune zerolatency -preset ultrafast -c:a aac -ar 48000
    -r 30
    -sc_threshold 0
    -force_key_frames expr:gte(t,n_forced*$3)
    -filter:v:0 scale=w=640:h=360:force_original_aspect_ratio=decrease -b:v:0 800k -maxrate 856k -b:a:0 96k
	-filter:v:1 scale=w=842:h=480:force_original_aspect_ratio=decrease -b:v:1 1400k -maxrate 1498k -b:a:1 128k
	-filter:v:2 scale=w=1280:h=720:force_original_aspect_ratio=decrease -b:v:2 2800k -maxrate 2996k -b:a:2 128k
	-var_stream_map $VAR_STREAM_MAP
    -hls_playlist 1 -streaming 1 -hls_time $3 -hls_list_size $4 -f hls
    -master_pl_name master.m3u8
    -method PUT -http_persistent 1 http://$2/upload/$1/%v.m3u8`

	args = strings.ReplaceAll(args, "$1", os.Args[1])
	args = strings.ReplaceAll(args, "$2", os.Args[2])
	args = strings.ReplaceAll(args, "$3", os.Args[3])
	args = strings.ReplaceAll(args, "$4", os.Args[4])

	argv := strings.Fields(args)
	for i, x := range argv {
		if x == "$VAR_STREAM_MAP" {
			argv[i] = "v:0,a:0,name:stream-360p v:1,a:1,name:stream-480p v:2,a:2,name:stream-720p"
		}
	}

	fmt.Println("CMDLINE", "ffmpeg", argv)

	cmd := exec.Command("/usr/bin/ffmpeg", argv...)
	r, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Error:", err)
		return 3
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		fmt.Println("Error:", err)
		return 4
	}

	fmt.Println("ID", os.Args[1], "START", cmd.Process.Pid)

	go func() {
		s := bufio.NewScanner(r)
		for s.Scan() {
			if ok, speed := parseSpeed(s.Text()); ok {
				stats := model.TranscoderStats{
					Id:                  os.Args[1],
					RealtimeCoefficient: speed,
				}

				fmt.Println("ID", os.Args[1], "SPEED", speed)

				var b bytes.Buffer
				if err := json.NewEncoder(&b).Encode(&stats); err != nil {
					fmt.Println("Warning:", err)
					continue
				}

				resp, err := http.Post(
					fmt.Sprintf("http://%s/transcoder/stats", os.Args[2]),
					"application/json",
					&b,
				)
				if err != nil {
					fmt.Println("Warning:", err)
					continue
				}
				resp.Body.Close()
			}
		}
	}()

	cmd.Wait()

	fmt.Println("ID", os.Args[1], "STOP", cmd.Process.Pid)

	return 0
}

func main() {
	os.Exit(run())
}
