FROM golang:latest as builder
WORKDIR /go/src/github.com/DBHeise/saucepan
ADD *.go ./
RUN go get .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o saucepan .


FROM scratch
WORKDIR /saucepan
ADD config.json ./
COPY --from=builder /go/src/github.com/DBHeise/saucepan/saucepan ./saucepan
CMD ["./saucepan", "--loglevel", "info"]