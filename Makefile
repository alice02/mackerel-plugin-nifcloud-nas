build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -o bin/mackerel-plugin-nifcloud-nas main.go
