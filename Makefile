all: mkcert/mkcert upcert/upcert getcert/getcert

mkcert/mkcert: mkcert/*.go
	go build -v -o mkcert/mkcert ./mkcert

upcert/upcert: upcert/*.go
	go build -v -o upcert/upcert ./upcert

getcert/getcert: getcert/*.go
	go build -v -o getcert/getcert ./getcert

update_getcert:
	GOOS=linux GOARCH=amd64 go build -v -o getcert/getcert ./getcert
	read -p "Enter user@host: " HOST && rsync --rsync-path="sudo rsync" --chmod=u+rwX,go-rwX -vPz getcert/getcert $$HOST:/usr/bin/getcert

clean:
	rm -f mkcert/mkcert upcert/upcert getcert/getcert
