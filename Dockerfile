FROM golang as build-stage
WORKDIR /app

COPY go.mod /app
COPY go.sum /app
RUN go mod download

ADD chat/ /app/chat
COPY main.go /app

RUN cd /app && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app

FROM alpine
COPY --from=build-stage /app/app /
CMD ["/app"]