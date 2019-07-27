# svck (pronounced 'service checker')

Check your http/https services.

## Install

### Binary Download

Navigate to [Releases](https://github.com/unprofession-al/svck/releases), grab
the package that matches your operating system and achitecture. Unpack the archive
and put the binary file somewhere in your `$PATH`

### From Source

Make sure you have [go](https://golang.org/doc/install) installed, then run: 

```
# go get -u https://github.com/unprofession-al/svck
```

## How to use

Define a bunch of services you want to check (`website.yaml`):

```yaml
---
my_website:
  addresses:
    - "very.unprofession.al"
  tests:
    root_no_ssl: 
      ssl: false
      resources:
        root:
          url:          "/"
          content_type: "text/html"
      status: 302
    very_important_pages:
      ssl: true
      resources:
        index:
          url:          "/"
          content_type: "text/html"
        very_important:
          url:          "bla.html"
          content_type: "text/html"
      status: 200
```

Then, run `svck`:

```
svck run website.yaml
Task (3/3)    0s [====================================================================] 100%
Failed check: my_website@very.unprofession.al/very_important_pages/very_important
	URL: https://very.unprofession.al/bla.html
	REQUEST_HEADERS: "X-Forwarded-Proto: https" "User-Agent: svck" "Host: very.unprofession.al"
	RESPONSE_HEADERS: "X-Content-Type-Options: nosniff" "X-Frame-Options: DENY" "Date: Sat, 27 Jul 2019 15:42:33 GMT" "Content-Type: text/plain; charset=utf-8" "Content-Length: 14" "Set-Cookie: __cfduid=dc314725d56b3bedead80614028a0ac801564242153; expires=Sun, 26-Jul-20 15:42:33 GMT; path=/; domain=.unprofession.al; HttpOnly" "Cache-Control: max-age=86400" "Vary: Accept-Encoding" "X-Xss-Protection: 1; mode=block" "Expect-Ct: max-age=604800, report-uri="https://report-uri.cloudflare.com/cdn-cgi/beacon/expect-ct"" "Server: cloudflare" "Strict-Transport-Security: max-age=31536000;" "Cf-Ray: 4fcfb9520c8bce97-GVA" 
	REASON: Expected 200, recieved 404

Summary:
	Failed: 1
	Successful: 2
```

## Credits

`svck` is based on an idea and a python/nose-based implmentation by @danduk82
