FROM golang:1.24-bookworm
RUN apt-get update && apt-get install -y --no-install-recommends \
      libgtk-4-1 xvfb ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /gtk4
COPY go.mod ./
COPY *.go ./
ENV GOFLAGS=-mod=mod CGO_ENABLED=0
RUN go mod tidy
CMD ["sh","-c","Xvfb :99 -screen 0 800x600x24 >/dev/null 2>&1 & sleep 1; DISPLAY=:99 GDK_BACKEND=x11 go test -count=1 -v -run TestLiveGTK4 ."]
