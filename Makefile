CGO_ENABLED=0
GOOS=linux
REGISTRYUI_HUB_URI=192.168.99.100:5000
REGISTRYUI_ACCOUNT_MGMT_ENABLED=true
REGISTRYUI_ACCOUNT_MGMT_CONFIG=./contrib/config/auth_config.yml
build:
			CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo .
docker-build: build
	docker build -t jgsqware/registry-ui .
ui-account:
			REGISTRYUI_HUB_URI=registry:5000 REGISTRYUI_ACCOUNT_MGMT_ENABLED=true REGISTRYUI_ACCOUNT_MGMT_CONFIG=./contrib/config/auth_config.yml REGISTRYUI_HYPERCLAIR_REPORT_ENABLED=true go run main.go
ui-account-docker:
			docker run --rm -v $(GOPATH):/go --net=contrib_default -e REGISTRYUI_HUB_URI=registry:5000 -e REGISTRYUI_ACCOUNT_MGMT_ENABLED=true -e REGISTRYUI_ACCOUNT_MGMT_CONFIG=./contrib/config/auth_config.yml -p 8080:8080 registry-ui-dev run main.go
