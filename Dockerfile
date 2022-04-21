FROM golang:alpine AS builder

RUN apk --update add ca-certificates git

RUN mkdir -p /go/src/github.com/lunemec/eve-accountant
WORKDIR /go/src/github.com/lunemec/eve-accountant
COPY . .

RUN go get github.com/ahmetb/govvv
RUN CGO_ENABLED=0 GOOS=linux govvv build -pkg github.com/lunemec/eve-accountant/pkg/version -o accountant

FROM scratch

# Port used for http server when running "login" command.
EXPOSE 3000

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/src/github.com/lunemec/eve-accountant/accountant .
ENTRYPOINT [ "/accountant" ]
