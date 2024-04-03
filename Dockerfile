FROM golang:1.22-alpine AS build

ENV PATH_ROJECT=${GOPATH}/src/github.com/tennuem/tbot
ENV APP=./cmd/app
ENV BIN=${GOPATH}/bin/tbot
ENV GO111MODULE=on

RUN apk --no-cache add git gcc libc-dev
WORKDIR ${PATH_ROJECT}
COPY . ${PATH_ROJECT}
RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -o ${BIN} ${APP}

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=build /go/bin/tbot /bin/tbot
ENTRYPOINT ["/bin/tbot"]
