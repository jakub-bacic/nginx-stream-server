FROM alpine:latest

ENV NGINX_VERSION=1.19.7

RUN apk update &&       \
    apk add             \
            git         \
            gcc         \
            binutils    \
            make        \
            ffmpeg      \
            g++         \
            pcre        \
            pcre-dev    \
            openssl     \
            openssl-dev \
            zlib-dev

WORKDIR /tmp

# RUN git clone https://github.com/nginx/nginx.git
RUN wget http://nginx.org/download/nginx-${NGINX_VERSION}.tar.gz  && \
    tar -xzf nginx-${NGINX_VERSION}.tar.gz                        && \
    rm -r nginx-${NGINX_VERSION}.tar.gz
RUN git clone https://github.com/arut/nginx-rtmp-module.git

RUN cd nginx-${NGINX_VERSION}                                           && \
    ./configure --add-module=../nginx-rtmp-module                       && \
    make	                                                            && \
    make install

COPY nginx.conf /usr/local/nginx/conf/nginx.conf
COPY docker-entrypoint.sh /
COPY transcoder.sh /usr/bin/

RUN mkdir /var/live

ENTRYPOINT ["/docker-entrypoint.sh"]

EXPOSE 1935
EXPOSE 80

STOPSIGNAL SIGQUIT

CMD ["/usr/local/nginx/sbin/nginx", "-g", "daemon off;"]