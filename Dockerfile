FROM golang:1.13 AS build

WORKDIR /usr/src/app

COPY go.mod go.sum ./

RUN go mod tidy

COPY . .

RUN make


FROM debian:10 AS app

COPY --from=build /usr/src/app/build/* /usr/sbin/hostsfile-daemon

ENTRYPOINT ["/usr/sbin/hostsfile-daemon"]
