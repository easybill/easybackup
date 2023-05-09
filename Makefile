run:
	go run main.go

build:
	go build easybackup

build-linux:
	GOOS=linux GOARCH=amd64 go build easybackup

install-mysql:
	brew install percona-server
	brew install percona-xtrabackup --force
	brew link --overwrite percona-xtrabackup

start-mysql:
	brew services restart percona-server