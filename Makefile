build:
	go build -o bin/app .

run: build
	sudo ./bin/app run alpine /bin/sh
ps:
	sudo ./bin/app ps
kill:
	sudo ./bin/app kill $(CONTAINER_ID)
build-scratch:
	go build -o bin/scratch ./scratch


scratch: build-scratch
	./bin/scratch

shell:
	limactl shell dev

shell-restart:
	limactl stop dev && \
	limactl start dev && \
	limactl shell dev

setup:
	mkdir -p rootfs images
	wget -O images/alpine.tar.gz https://dl-cdn.alpinelinux.org/alpine/v3.23/releases/aarch64/alpine-minirootfs-3.23.4-aarch64.tar.gz
	tar -C rootfs -xzf images/alpine.tar.gz 

# run on lima side
setup-linux:
	mkdir -p ~/rootfs
	tar -C ~/rootfs -xzf images/alpine.tar.gz

clean-linux:
	rm -f bin/app
	sudo unmount -l /home/peterbull.guest/rootfs/proc 2>/dev/null || true
	sudo unmount -l /home/peterbull.guest/rootfs 2>/dev/null || true
