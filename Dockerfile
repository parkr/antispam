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
FROM debian:13-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates

RUN update-ca-certificates
WORKDIR /app/
RUN touch /tmp/antispam-filter.json
COPY --from=builder /go/bin/antispam /bin/antispam
ENTRYPOINT ["/bin/antispam"]
