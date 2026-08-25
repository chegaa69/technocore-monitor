# Build
FROM golang:1.21-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /technocore-monitor .

# Run
FROM gcr.io/distroless/static-debian12
COPY --from=build /technocore-monitor /technocore-monitor
EXPOSE 9184
ENTRYPOINT ["/technocore-monitor"]
