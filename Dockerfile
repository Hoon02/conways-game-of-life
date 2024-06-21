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
    pulseaudio \
    pulseaudio-utils \
    libpulse-dev \
    git \
    golang-go

# ffmpeg 설치
RUN apt-get install -y ffmpeg

WORKDIR /app
COPY . .
RUN go mod tidy && \
    go mod download

RUN go get github.com/go-gl/gl/v2.1/gl && \
    go get github.com/go-gl/glfw/v3.2/glfw && \
    go get github.com/mesilliac/pulse-simple

RUN go build -v -o nesexe

ENV DISPLAY=:1
ENV PULSE_LATENCY_MSEC=1

RUN echo "#!/bin/bash\n\
pulseaudio -D --exit-idle-time=-1 &\n\
sleep 5\n\
pacmd load-module module-null-sink sink_name=v1 rate=44100 channels=1\n\
pacmd set-default-sink v1\n\
pacmd set-default-source v1.monitor" > pulseaudio-setup.sh && \
chmod +x pulseaudio-setup.sh

# ffmpeg에서 GPU 가속 사용을 위한 옵션 추가
CMD ["bash", "-c", "./pulseaudio-setup.sh && Xvfb :1 -screen 0 400x400x24 & sleep 10 && DISPLAY=:1 go run main.go && ffmpeg -f pulse -i default -f x11grab -s 400x400 -i :1 -map 0:a:0 -c:a libopus -compression_level 0 -b:a 24k -af aresample=async=0.5 -map 1:v:0 -r 60 -c:v libx264 -preset ultrafast -tune zerolatency -b:v 1000k -f rtsp rtsp://mtx:8554/mystream"]
EXPOSE 8080