# certutils

- `mkcert` Generate wildcard SSL certificates automatically. It helps you set up TXT DNS records on Alidns or Cloudflare.
- `upcert` Upload and encrypt cert files to Aliyun OSS.
- `getcert` Download and decrypt encrypted cert files on Aliyun OSS.

You can run `go run generate_key.go` to generate `key.go` for upcert and getcert.

## mkcert

Make sure you have installed:

- [aliyun-cli](https://github.com/aliyun/aliyun-cli) and/or [cloudflare](https://github.com/caiguanhao/cloudflare)
- docker
- docker pull certbot/certbot:v1.10.0

Note: You may be [rate-limited](https://letsencrypt.org/docs/rate-limits/) if you are going to make many certs with the same IP address.

## Usage

![certutils](https://user-images.githubusercontent.com/1284703/112626352-0ca95180-8e6b-11eb-8eeb-c55930fc1efa.gif)

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

➜ getcert
2021/01/04 11:08:40 getting list of certs
1. example.com{.cert,.key}       4. foobar.com{.cert,.key}        7. helloworld.com{.cert,.key}
2. example.net{.cert,.key}       5. foobar.net{.cert,.key}
3. example.org{.cert,.key}       6. foobar.org{.cert,.key}
Enter numbers (separated by comma) to choose files: 1
2021/01/04 11:08:58 downloading example.com.cert
2021/01/04 11:08:58 written example.com.cert
2021/01/04 11:08:58 downloading example.com.key
2021/01/04 11:08:58 written example.com.key
```
