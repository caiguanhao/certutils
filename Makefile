all: mkcert/mkcert upcert/upcert getcert/getcert

mkcert/mkcert: mkcert/*.go
	(cd mkcert && go build -v -o mkcert)

upcert/upcert: upcert/*.go
	(cd upcert && go build -v -o upcert)

getcert/getcert: getcert/*.go
	(cd getcert && go build -v -o getcert)

update_getcert:
	GOOS=linux GOARCH=amd64 go build -v -o getcert/getcert ./getcert
	read -p "Enter user@host: " HOST && rsync --rsync-path="sudo rsync" --chmod=u+rwX,go-rwX -vPz getcert/getcert $$HOST:/usr/bin/getcert

clean:
	rm -f mkcert/mkcert upcert/upcert getcert/getcert
