FROM golang:1.16-alpine AS builder
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/go-comfoair

FROM scratch
COPY --from=builder /out/go-comfoair /go-comfoair

ENTRYPOINT ["/go-domestia"]