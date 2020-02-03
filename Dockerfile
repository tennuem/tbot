FROM golang:1.13-alpine AS build

ENV PATH_ROJECT=${GOPATH}/src/github.com/tennuem/tbot
ENV APP=./cmd/app
ENV BIN=${GOPATH}/bin/tbot
ENV GO111MODULE=on

RUN apk add --no-cache git
WORKDIR ${PATH_ROJECT}
COPY . ${PATH_ROJECT}
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o ${BIN} ${APP}

FROM alpine:3.10
RUN apk --no-cache add ca-certificates
COPY --from=build /go/bin/tbot /bin/tbot
ENTRYPOINT ["/bin/tbot"]
