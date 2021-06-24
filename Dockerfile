# First, build
FROM golang as builder
WORKDIR /go/src/github.com/parkr/antispam
COPY statik statik
COPY go* ./
COPY *.go ./
RUN ls
RUN go install github.com/parkr/antispam/...
RUN go test github.com/parkr/antispam/...

# Then, package
FROM scratch
COPY --from=builder /go/bin/antispam /bin/antispam
ENTRYPOINT ["/bin/antispam"]
