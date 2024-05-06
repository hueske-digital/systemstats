FROM golang:1.22-alpine as builder

WORKDIR /app

COPY go.mod go.sum main.go ./

RUN apk add --no-cache ca-certificates

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

FROM scratch

WORKDIR /root/

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]

