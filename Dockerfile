FROM golang:1.13.4-alpine AS builder

WORKDIR /go/src/github.com/aokumasan/mackerel-plugin-nifcloud-nas
RUN apk add --no-cache make
ADD . .
RUN make build


FROM mackerel/mackerel-agent:0.64.0

COPY --from=builder /go/src/github.com/aokumasan/mackerel-plugin-nifcloud-nas/bin/mackerel-plugin-nifcloud-nas /bin/mackerel-plugin-nifcloud-nas
