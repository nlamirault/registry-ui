FROM golang
WORKDIR /go/src/github.com/jgsqware/registry-ui
EXPOSE 8080
ENTRYPOINT ["go","run","main.go"]
