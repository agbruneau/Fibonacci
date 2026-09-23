# FibCalc container build image.
#
# Not "reproducible": the base images are pinned by digest (see each FROM), but
# `apt-get install make` in the builder takes whatever the Debian archive serves
# that day, and BUILD_DATE is compiled into the binary. Fixed base, not
# reproducible artifact — the header claimed the stronger property until
# 2026-09-07.
#
# Two-stage build:
#  1. builder — CGO-disabled static build of the default binary (no `gmp`
#     build tag). Profile-guided optimisation consumes
#     cmd/fibcalc/default.pgo when present.
#  2. runtime — distroless minimal base shipping only the linked binary.
#
# Build responsibility is delegated to the Makefile (audit STR-02): the compile
# flags — -trimpath, the PGO profile, and the three -X version symbols — live in
# exactly one place. This image used to repeat a bare `go build -ldflags="-s -w"`
# that omitted the version symbols entirely, so `docker run ... --version`
# reported "fibcalc dev / Commit: unknown / Built: unknown".
#
# .dockerignore excludes .git, so `git describe` cannot run inside the build.
# Pass the identity in explicitly; the defaults below are honest placeholders,
# not guesses at a version:
#
#   docker build \
#     --build-arg VERSION="$(git describe --tags --always --dirty)" \
#     --build-arg COMMIT="$(git rev-parse --short HEAD)" \
#     --build-arg BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
#     -t fibcalc:local .

# Base pinned by digest (SEC-04, closed 2026-09-07). When a reference carries
# both a tag and a digest it is fetched by digest and the tag is never checked
# against it: the tag is a label for humans, the digest is the identity. Bump
# both together.
#
# Provenance of the value, which is the whole point of it being here: resolved
# by the CI `docker` job, run 34135480664, step "base image digests", which ran
#     docker buildx imagetools inspect golang:1.26-bookworm --format '{{.Manifest.Digest}}'
# on a GitHub runner. This is the multi-arch INDEX digest — the same run printed
# MediaType `application/vnd.oci.image.index.v1+json` and listed linux/amd64,
# arm/v7, arm64/v8, 386, ppc64le and s390x under it. A single platform's manifest
# digest would still build green on the amd64 runner, so nothing in CI would
# catch that mistake; the MediaType line next to the digest is the check.
#
# The pin stayed open through four audits for one reason: no environment this
# repository had been edited from had a docker CLI or outbound registry access,
# and a guessed digest is worse than a tag. The CI job added by the 2026-09-07
# audit is the first environment with both.
#
# What the pin buys: an upstream re-push of golang:1.26-bookworm can no longer
# change the toolchain under an unchanged source tree, and a version bump becomes
# a readable commit instead of silent drift.
# What it does not buy: a reproducible build. The `apt-get install make` below
# still takes whatever the Debian archive serves that day, and BUILD_DATE is
# compiled into the binary. This is a fixed base, not a reproducible artifact.
#
# ARG GO_VERSION is gone with the tag: sitting next to a digest it would select
# nothing, so it could only mislead.
FROM golang:1.26-bookworm@sha256:9fdc884aacc3bec89b20ffc69f4bb369c78210e3e4f600387b5128b12c199f81 AS builder

ENV CGO_ENABLED=0 \
    GOFLAGS=-trimpath

WORKDIR /src

# The Makefile is the build contract; golang:*-bookworm ships git but not make.
RUN apt-get update \
    && apt-get install -y --no-install-recommends make \
    && rm -rf /var/lib/apt/lists/*

# Cache module downloads independently of source edits.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Identity of the build, supplied by the caller (see header). The Makefile reads
# these as overridable variables and injects them via -X.
ARG VERSION=docker
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# `make build` applies -trimpath, the -X version symbols and the PGO profile
# from cmd/fibcalc/default.pgo when it is present. The --version call is a smoke
# test: it fails the build if the binary cannot start.
RUN make build VERSION="${VERSION}" COMMIT="${COMMIT}" BUILD_DATE="${BUILD_DATE}" \
    && mkdir -p /out \
    && cp build/fibcalc /out/fibcalc \
    && /out/fibcalc --version


# Same rule as the builder stage: index digest, tag decorative. Same provenance —
# run 34135480664 printed MediaType `application/vnd.oci.image.index.v1+json`
# with linux/amd64, arm64/v8, arm/v7, s390x and ppc64le beneath it.
#
# A wrong pin here would NOT fail the build. BuildKit resolves the metadata of
# every stage before running anything, and it applies a platform matcher only
# when the descriptor is an index; pointed at a single manifest it takes it as
# given. The failure would be an arm64 rootfs holding an amd64 binary, surfacing
# at `docker run` on someone else's machine — which is why the MediaType check
# above is done by eye at paste time and not by a CI gate.
#
# This is the plain tag's digest, not the `:nonroot` variant's: that variant only
# changes the default USER, and the explicit USER line below already does that.
#
# Nothing in this repository will refresh this line. There is no Dependabot or
# Renovate configuration, and govulncheck reads the Go module graph, not this
# layer. The pin trades silent update for silent staleness; the only signal is
# comparing this value against what the CI job prints.
FROM gcr.io/distroless/base-debian12@sha256:fabbf1c0c357a3d42550111351daed089b20a2c954df13ee2fcff60602515e84 AS runtime

COPY --from=builder /out/fibcalc /usr/local/bin/fibcalc
# The bigfft-derived code is BSD-3-Clause, whose clause 2 requires its notice to
# travel with any binary distribution; NOTICE carries it in full (EVAL-01).
COPY --from=builder /src/LICENSE /src/NOTICE /usr/share/doc/fibcalc/

USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/fibcalc"]
CMD ["--help"]
