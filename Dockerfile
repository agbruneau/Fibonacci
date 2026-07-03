# FibCalc reproducible build image.
#
# Two-stage build:
#  1. builder — CGO-disabled static build of the default binary (no `gmp`
#     build tag). Profile-guided optimisation consumes
#     cmd/fibcalc/default.pgo when present.
#  2. runtime — distroless minimal base shipping only the linked binary.
#
# Audit-PRD E6 / Sprint S1-T6.

ARG GO_VERSION=1.26

# TODO(SEC-04): pin by digest (golang:${GO_VERSION}-bookworm@sha256:...).
# Not pinned here because this sandbox has no verified registry access
# (docker CLI absent, registry auth calls blocked) to resolve a trustworthy
# digest — do not guess one. Resolve with `docker buildx imagetools inspect
# golang:1.26-bookworm` or `crane digest golang:1.26-bookworm` on a machine
# with registry access, then paste the sha256 here.
FROM golang:${GO_VERSION}-bookworm AS builder

ENV CGO_ENABLED=0 \
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


# TODO(SEC-04): pin by digest (gcr.io/distroless/base-debian12@sha256:...).
# Same reason as the builder stage above — resolve with
# `crane digest gcr.io/distroless/base-debian12` on a machine with registry
# access, then paste the sha256 here.
FROM gcr.io/distroless/base-debian12 AS runtime

COPY --from=builder /out/fibcalc /usr/local/bin/fibcalc

USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/fibcalc"]
CMD ["--help"]
