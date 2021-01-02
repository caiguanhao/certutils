all: mkcert/mkcert upcert/upcert getcert/getcert

mkcert/mkcert: mkcert/*.go
	go build -v -o mkcert/mkcert ./mkcert

upcert/upcert: upcert/*.go
	go build -v -o upcert/upcert ./upcert

getcert/getcert: getcert/*.go
	go build -v -o getcert/getcert ./getcert

clean:
	rm -f mkcert/mkcert upcert/upcert getcert/getcert
