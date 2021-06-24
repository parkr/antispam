# First, build
FROM golang as builder
WORKDIR /app/antispam
COPY statik statik
COPY go* ./
COPY *.go ./
RUN ls -l \
  && go install ./... \
  && go test ./... \
  && ls -lh

# Then, package
FROM debian:buster-slim
COPY --from=builder /go/bin/antispam /bin/antispam
ENTRYPOINT ["/bin/antispam"]
