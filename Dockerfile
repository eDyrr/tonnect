FROM golang:1.25.0-alpine AS build
WORKDIR /.
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /tonnect .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /tonnect /tonnect
EXPOSE 8080
CMD ["/tonnect"]