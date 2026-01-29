FROM golang:1.25 AS builder

ARG TARGET_ARCH

WORKDIR /app

RUN go install github.com/a-h/templ/cmd/templ@v0.3.977

RUN wget -O /usr/local/bin/tailwindcss https://github.com/tailwindlabs/tailwindcss/releases/download/v4.1.18/tailwindcss-linux-x64
RUN chmod +x /usr/local/bin/tailwindcss

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN mkdir -p tmp/build

RUN templ generate
RUN tailwindcss -i static/css/app.css -o static/css/out.css --minify

RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGET_ARCH} go build -ldflags "-w -s" -o tmp/build/app cmd/server/main.go
RUN chmod +x tmp/build/app

FROM scratch

WORKDIR /app

COPY --from=builder /app/tmp/build/app /app/app

CMD ["/app/app"]
