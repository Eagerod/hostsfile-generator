FROM golang:1.18 AS build

WORKDIR /usr/src/app

COPY go.mod go.sum ./

RUN go mod download

ARG VERSION UnspecifiedContainerVersion

COPY . .

RUN make
RUN make test


FROM debian:10 AS app

COPY --from=build /usr/src/app/build/* /usr/sbin/hostsfile-daemon

ENTRYPOINT ["/usr/sbin/hostsfile-daemon"]
