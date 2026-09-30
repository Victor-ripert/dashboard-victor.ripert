FROM golang:1.22-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o gateway .

FROM alpine:3.20

COPY --from=build /app/gateway /gateway

EXPOSE 8080

CMD ["/gateway"]
