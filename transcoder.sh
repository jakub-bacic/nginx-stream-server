#!/bin/sh
# ffmpeg -re -i rtmp://localhost:1935/live/$1 \
#     -max_muxing_queue_size 9999 \
#     -map 0:v:0 -map 0:a:0 -map 0:v:0 -map 0:a:0 -map 0:v:0 -map 0:a:0 -map 0:v:0 -map 0:a:0 \
#     -c:v libx264 -tune zerolatency -preset ultrafast -c:a aac -ar 48000 \
#     -sc_threshold 0 \
#     -force_key_frames "expr:gte(t,n_forced*1)" \
#     -filter:v:0 scale=w=640:h=360:force_original_aspect_ratio=decrease -b:v:0 800k -maxrate 856k -bufsize 1200k -b:a:0 96k \
#     -filter:v:1 scale=w=842:h=480:force_original_aspect_ratio=decrease -b:v:1 1400k -maxrate 1498k -bufsize 2100k -b:a:1 128k \
#     -filter:v:2 scale=w=1280:h=720:force_original_aspect_ratio=decrease -b:v:2 2800k -maxrate 2996k -bufsize 4200k -b:a:2 128k \
#     -filter:v:3 scale=w=1920:h=1080:force_original_aspect_ratio=decrease -b:v:3 5000k -maxrate 5350k -bufsize 7500k -b:a:3 192k \
#     -var_stream_map "v:0,a:0,name:360p v:1,a:1,name:480p v:2,a:2,name:720p v:3,a:3,name:1080p" \
#     -hls_flags independent_segments -lhls 1 -hls_time 1 -hls_list_size 4 -f hls -hls_playlist_type event \
#     -master_pl_name "stream-master.m3u8" \
#     -method PUT -http_persistent 1 "http://gcsproxy:8081/upload/$1/stream-%v.m3u8"

ffmpeg -re -i rtmp://localhost:1935/live/$1 \
    -max_muxing_queue_size 9999 \
    -map 0:v:0 -map 0:a:0 -map 0:v:0 -map 0:a:0 \
    -c:v libx264 -tune zerolatency -preset ultrafast -tune zerolatency -c:a aac -ar 48000 \
    -sc_threshold 0 \
    -force_key_frames "expr:gte(t,n_forced*2)" \
    -filter:v:0 scale=w=640:h=360:force_original_aspect_ratio=decrease -b:v:0 800k -maxrate 856k -bufsize 1200k -b:a:0 96k \
    -filter:v:2 scale=w=1280:h=720:force_original_aspect_ratio=decrease -b:v:1 2800k -maxrate 2996k -bufsize 4200k -b:a:1 128k \
    -var_stream_map "v:0,a:0,name:steam-360p v:1,a:1,name:stream-720p" \
    -hls_playlist 1 -streaming 1 -hls_time 2 -hls_list_size 2 -f hls \
    -master_pl_name "master.m3u8" \
    -master_pl_publish_rate 1 \
    -method PUT -http_persistent 1 "http://gcsproxy:8081/upload/$1/%v.m3u8"

# ffmpeg -re -i rtmp://localhost:1935/live/$1 \
#     -max_muxing_queue_size 9999 \
#     -map 0:v:0 -map 0:a:0 \
#     -c:v libx264 -tune zerolatency -preset ultrafast -c:a aac -ar 48000 \
#     -sc_threshold 0 \
#     -filter:v:0 scale=w=640:h=360:force_original_aspect_ratio=decrease -b:v:0 800k -maxrate 856k -bufsize 1200k -b:a:0 96k \
#     -filter:v:2 scale=w=1280:h=720:force_original_aspect_ratio=decrease -b:v:1 2800k -maxrate 2996k -bufsize 4200k -b:a:1 128k \
#     -var_stream_map "v:0,a:0 v:1,a:1" \
#     -bf 1 \
#     -keyint_min 24 -g 24 -sc_threshold 0 -b_strategy 0 -ar:a:1 48000 -use_timeline 1 -use_template 1 \
#     -window_size 4 -adaptation_sets "id=0,streams=v id=1,streams=a" \
#     -hls_playlist 1 -seg_duration 2 -streaming 1 -strict experimental -remove_at_exit 1 \
#     -target_latency 4 \
#     -method PUT -http_persistent 1 "http://gcsproxy:8081/upload/$1/stream.mpd"

# ffmpeg -re -i rtmp://localhost:1935/live/$1 \
#     -max_muxing_queue_size 9999 \
#     -map 0 -map 0 -map 0 -c:a aac -c:v libx264 -tune zerolatency -preset ultrafast \
#     -b:v:0 2800k -s:v:0 1280x720 -profile:v:0 high \
#     -b:v:1 800k -s:v:1 640x360   -profile:v:1 main \
#     -b:v:2 500k -s:v:2 320x240   -profile:v:2 baseline \
#     -bf 1 \
#     -keyint_min 24 -g 24 -sc_threshold 0 -b_strategy 0 -ar:a:1 48000 -use_timeline 1 -use_template 1 \
#     -window_size 5 -adaptation_sets "id=0,streams=v id=1,streams=a" \
#     -hls_playlist 1 -seg_duration 1 -streaming 1  -strict experimental -lhls 1 -remove_at_exit 1 \
#     -method PUT -http_persistent 1 "http://gcsproxy:8081/upload/$1/stream.mpd"
