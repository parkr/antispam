# First, build
FROM golang as builder
WORKDIR /go/src/github.com/parkr/antispam
COPY vendor vendor
COPY statik statik
COPY *.go ./
RUN ls
RUN CGO_ENABLED=0 GOOS=linux go install github.com/parkr/antispam/...
RUN CGO_ENABLED=0 GOOS=linux go test github.com/parkr/antispam/...

# Then, package
FROM scratch
COPY --from=builder /go/bin/antispam /bin/antispam
ENTRYPOINT ["/bin/antispam"]
