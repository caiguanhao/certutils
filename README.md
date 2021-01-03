# certutils

- `mkcert` Generate wildcard SSL certificates and update Aliyun DNS records automatically.
- `upcert` Upload and encrypt cert files to OSS.
- `getcert` Download and decrypt encrypted cert files on OSS.

You can run `go run generate_key.go` to generate `key.go` for upcert and getcert.

## mkcert

You must have installed:

- [aliyun-cli](https://github.com/aliyun/aliyun-cli) and/or [cloudflare](https://github.com/caiguanhao/cloudflare)
- docker
- docker pull certbot/certbot

## Usage

```
➜ mkcert "*.example.com"
2021/01/04 02:21:39 root domain: example.com
2021/01/04 02:21:39 created container: e7378c26
2021/01/04 02:21:39 finding TXT records for _acme-challenge
2021/01/04 02:21:39 found 2 TXT records for _acme-challenge
2021/01/04 02:21:39 deleting TXT record with id 18862666777171968
2021/01/04 02:21:40 deleting TXT record with id 18862665550077952
2021/01/04 02:21:41 waiting acme challenge
2021/01/04 02:21:45 pressing enter to certbot, waiting for response...
2021/01/04 02:21:45 received certbot's acme challenge: ubRcq6JSoXynolCWf1TT2nhUlQwEok3Lmig1gryr65c
2021/01/04 02:21:45 creating new TXT record
2021/01/04 02:21:45 new record has been created, id: 21028086258404352
2021/01/04 02:21:45 received certbot's acme challenge: KNHNcYb6fWYw6SdvWTxxmP-ybPfYGt6iLi6jSLia26g
2021/01/04 02:21:45 creating new TXT record
2021/01/04 02:21:46 new record has been created, id: 21028086297198592
2021/01/04 02:21:46 wait 10 seconds for dns records to take effect
2021/01/04 02:21:56 waiting cert files
2021/01/04 02:21:56 pressing enter to certbot, waiting for response...
2021/01/04 02:22:04 copying /etc/letsencrypt/live/example.com/fullchain.pem from e7378c26
2021/01/04 02:22:04 written file example.com.cert
2021/01/04 02:22:04 copying /etc/letsencrypt/live/example.com/privkey.pem from e7378c26
2021/01/04 02:22:04 written file example.com.key
2021/01/04 02:22:04 successfully generated certificates
2021/01/04 02:22:04 removing container e7378c26
2021/01/04 02:22:04 done
➜ upcert example.com.*
2021/01/04 02:22:23 uploaded example.com.cert
2021/01/04 02:22:23 uploaded example.com.key
```
