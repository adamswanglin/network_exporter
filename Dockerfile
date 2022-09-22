FROM golang:alpine as builder
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o network_exporter .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/network_exporter network_exporter
COPY --from=builder /app/network_exporter.yml /app/network_exporter.yml
CMD /app/network_exporter
EXPOSE 9427
