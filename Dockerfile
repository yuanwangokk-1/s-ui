FROM --platform=$BUILDPLATFORM node:alpine AS front-builder
WORKDIR /app
COPY frontend/ ./
RUN npm install && npm run build

FROM golang:1.23-alpine AS backend-builder
WORKDIR /app
ARG TARGETARCH
ENV CGO_CFLAGS="-D_LARGEFILE64_SOURCE"
ENV CGO_ENABLED=1
ENV GOARCH=$TARGETARCH
RUN apk update && apk --no-cache --update add build-base gcc wget unzip
COPY . .
COPY --from=front-builder  /app/dist/ /app/web/html/
RUN go build -ldflags="-w -s" -tags "with_quic,with_grpc,with_ech,with_utls,with_reality_server,with_acme,with_gvisor" -o sui main.go

FROM --platform=$TARGETPLATFORM alpine
LABEL org.opencontainers.image.authors="alireza7@gmail.com"
ENV TZ=Asia/Tehran
WORKDIR /app
RUN apk add  --no-cache --update ca-certificates tzdata
COPY --from=backend-builder  /app/sui /app/
COPY entrypoint.sh /app/
VOLUME [ "s-ui" ]
ENTRYPOINT [ "./entrypoint.sh" ]