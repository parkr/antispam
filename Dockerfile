# First, build
FROM golang:1.10.0 as builder
WORKDIR /go/src/github.com/parkr/antispam
ADD . .
RUN go version
RUN CGO_ENABLED=0 GOOS=linux go install github.com/parkr/antispam/...

FROM scratch
COPY --from=builder /go/bin/antispam /bin/antispam
CMD ["/bin/antispam"]
