FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    libgl1-mesa-dev \
    libx11-dev \
    libxcursor-dev \
    libxrandr-dev \
    libxinerama-dev \
    libxi-dev \
    libxxf86vm-dev \
    pkg-config \
    xvfb \
    git \
    golang-go \
    ffmpeg

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go mod tidy
RUN go build -o conways-game-of-life

ENV DISPLAY=:1

CMD ["bash", "-c", "Xvfb :1 -screen 0 800x800x24 & sleep 10 && DISPLAY=:1 ./conways-game-of-life & sleep 10 && ffmpeg -f x11grab -draw_mouse 0 -s 800x800 -i :1 -r 154 -c:v libx264 -preset ultrafast -tune zerolatency -b:v 2000k -f rtsp rtsp://mtx:8554/mystream"]

EXPOSE 8080