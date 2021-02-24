#!/bin/sh

set -e 

sed -i "s|{{PUBLISH_PASSWORD}}|${PUBLISH_PASSWORD}|g" /usr/local/nginx/conf/nginx.conf

exec "$@"