ARG PROXY_REGISTRY=

FROM ${PROXY_REGISTRY:+$PROXY_REGISTRY/}golang:1.15.3-alpine3.12 as builder

ARG VERSION

COPY . /go/src/keycloak-operator/
WORKDIR /go/src/keycloak-operator/cmd/keycloak-operator

ENV CGO_ENABLED=0 \
    GO111MODULE=on

RUN go install

FROM ${PROXY_REGISTRY:+$PROXY_REGISTRY/}alpine:3.12.1 as runtime


COPY --from=builder /go/bin/keycloak-operator /keycloak-operator

RUN adduser -S appuser -u 1000 -G root
USER 1000

ENTRYPOINT ["/keycloak-operator"]
