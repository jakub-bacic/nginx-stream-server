#!/bin/bash
ffmpeg -f avfoundation -video_size 1280x720 -framerate 30 -i "0:0" -vcodec libx264 -tune zerolatency -pix_fmt yuv420p -f flv rtmp://localhost:1935/live/facetime