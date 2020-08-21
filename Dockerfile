FROM golang:alpine as builder
WORKDIR /main
COPY . /main
RUN go build 

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /main/prometheus-scc-metrics /main
EXPOSE 8080
CMD ["/main"]