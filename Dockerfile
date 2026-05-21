# FibCalc reproducible build image.
#
# Two-stage build:
#  1. builder — toolchain + CGO + libgmp-dev to compile with the `gmp`
#     build tag and to run `go test -race`. Profile-guided optimisation
#     consumes cmd/fibcalc/default.pgo when present.
#  2. runtime — distroless minimal base shipping only the linked binary.
#
# Audit-PRD E6 / Sprint S1-T6.

ARG GO_VERSION=1.26

FROM golang:${GO_VERSION}-bookworm AS builder

# CGO toolchain + libgmp headers required by the optional `gmp` backend
# and by `go test -race`.
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        build-essential \
        libgmp-dev \
    && rm -rf /var/lib/apt/lists/*

ENV CGO_ENABLED=1 \
    GOFLAGS=-trimpath

WORKDIR /src

# Cache module downloads independently of source edits.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the static-friendly binary. PGO profile is consumed automatically
# when cmd/fibcalc/default.pgo exists in the source tree.
RUN go build -ldflags="-s -w" -o /out/fibcalc ./cmd/fibcalc \
    && /out/fibcalc --version > /dev/null


FROM gcr.io/distroless/base-debian12 AS runtime

COPY --from=builder /out/fibcalc /usr/local/bin/fibcalc

USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/fibcalc"]
CMD ["--help"]
