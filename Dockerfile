# Etapa de compilación
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod ./
COPY main.go ./
RUN go build -o api main.go

# Etapa final (imagen liviana)
FROM alpine:latest
WORKDIR /app
COPY --from=build /app/api .
EXPOSE 8080
ENTRYPOINT ["./api"]
