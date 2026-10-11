FROM golang:1.22-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o github-service .

FROM alpine:3.20
RUN apk add --no-cache wget
COPY --from=build /app/github-service /github-service
EXPOSE 4000
CMD ["/github-service"]
