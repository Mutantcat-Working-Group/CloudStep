# CloudStep 容器镜像：多阶段构建，产出 CGO_ENABLED=0 静态二进制。
# 数据库与运行数据写在 /data，建议挂载卷持久化。
FROM node:22-alpine AS web

WORKDIR /web

# Web 资源不打进仓库（web/cloud-step-web-1g/.gitignore 忽略 dist），
# 但 Go 侧 go:embed 需要它存在，因此先构建前端。
COPY web/cloud-step-web-1g/package.json web/cloud-step-web-1g/yarn.lock ./
RUN yarn install --frozen-lockfile

COPY web/cloud-step-web-1g ./
RUN yarn build

FROM golang:1.25-alpine AS build

ARG VERSION=1.0.20260920

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 使用镜像内新构建的前端产物，避免不同机器本地 dist 不一致。
COPY --from=web /web/dist web/cloud-step-web-1g/dist

RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/cloud-step .

FROM alpine:3.21

RUN adduser -D -H -u 10001 cloudstep \
    && mkdir -p /data \
    && chown cloudstep:cloudstep /data

COPY --from=build /out/cloud-step /usr/local/bin/cloud-step

USER cloudstep
WORKDIR /data
VOLUME ["/data"]
EXPOSE 9091

ENTRYPOINT ["/usr/local/bin/cloud-step"]
