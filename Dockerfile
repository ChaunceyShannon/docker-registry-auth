FROM golang:1.17.1-buster as golang-builder

ARG BIN_NAME=run

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
COPY *.go ./

RUN go mod tidy
RUN CGO_ENABLED=0 go build -o ${BIN_NAME}

FROM chaunceyshannon/cicd-tools:1.0.0 as upx-builder

ARG BIN_NAME=run

WORKDIR /app

COPY --from=golang-builder /app/${BIN_NAME} ./

RUN upx -9 ${BIN_NAME}

FROM registry:2

ARG BIN_NAME=run
ENV BIN_NAME=${BIN_NAME}

COPY --from=upx-builder /app/${BIN_NAME} /bin/${BIN_NAME}

ENTRYPOINT []

CMD /bin/${BIN_NAME}

# FROM registry:2 as registry-image
#FROM ubuntu:20.04

# ARG BIN_NAME=run
# ENV BIN_NAME=${BIN_NAME}

# COPY --from=registry-image /entrypoint.sh /entrypoint.sh
# COPY --from=registry-image /bin/registry /bin/registry
# COPY --from=registry-image /etc/docker/registry/config.yml /etc/docker/registry/config.yml

# COPY --from=upx-builder /app/${BIN_NAME} /bin/${BIN_NAME}

# ENTRYPOINT []

# CMD /bin/${BIN_NAME}
