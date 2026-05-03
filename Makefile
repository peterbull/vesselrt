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

	
