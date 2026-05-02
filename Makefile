build:
	go build -o bin/app .

run: build
	sudo ./bin/app run ubuntu /bin/sh

shell:
	limactl shell dev
