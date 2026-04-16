FROM golang:1.24-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 go build -o /bin/gophermart ./cmd/gophermart

FROM alpine:3.20
RUN adduser -D app
USER app
COPY --from=build /bin/gophermart /bin/gophermart
EXPOSE 8080
ENTRYPOINT ["/bin/gophermart"]
