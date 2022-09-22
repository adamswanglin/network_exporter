FROM golang:alpine as builder
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o network_exporter .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
RUN addgroup  -g 1002 non-root && adduser  -s /sbin/nologin --disabled-password --no-create-home --uid 1001 --ingroup non-root non-root
WORKDIR /app
COPY --from=builder --chown=non-root:non-root /app/network_exporter network_exporter
COPY --from=builder --chown=non-root:non-root /app/network_exporter.yml /app/network_exporter.yml
USER non-root
CMD /app/network_exporter
EXPOSE 9427
