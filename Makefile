build:
	go build -o bin/app .

run: build
	sudo ./bin/app run ubuntu /bin/sh

build-scratch:
	go build -o bin/scratch ./scratch


scratch: build-scratch
	./bin/scratch

shell:
	limactl shell dev

setup:
	mkdir -p rootfs images
	wget -O images/alpine.tar.gz https://dl-cdn.alpinelinux.org/alpine/v3.23/releases/aarch64/alpine-minirootfs-3.23.4-aarch64.tar.gz
	tar -C rootfs -xzf images/alpine.tar.gz 
