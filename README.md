# certutils

- `mkcert` Generate wildcard SSL certificates and update Aliyun DNS records automatically.
- `upcert` Upload and encrypt cert files to OSS.
- `getcert` Download and decrypt encrypted cert files on OSS.

You can run `go run generate_key.go` to generate `key.go` for upcert and getcert.

## mkcert

You must have installed:

- [aliyun-cli](https://github.com/aliyun/aliyun-cli)
- docker
- docker pull certbot/certbot
