FROM golang:1.10-alpine as build-env
WORKDIR /go/src/github.com/gilgameshskytrooper/bigdisk/
RUN apk --no-cache add ca-certificates && apk --no-cache add git
COPY main.go .
COPY /email ./email
COPY /crypto ./crypto
COPY /sort ./sort
COPY /structs ./structs
COPY /utils ./utils
COPY /app ./app
RUN go get -d -v ./...
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bigdisk .

FROM scratch
COPY /public /public
COPY /templates /templates
COPY --from=build-env /tmp /tmp
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-env /go/src/github.com/gilgameshskytrooper/bigdisk/bigdisk /
CMD ["./bigdisk"]
