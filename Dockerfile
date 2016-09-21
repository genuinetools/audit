FROM alpine:latest
MAINTAINER Jessica Frazelle <jess@linux.com>

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

RUN	apk add --no-cache \
	ca-certificates

COPY . /go/src/github.com/jfrazelle/audit

RUN set -x \
	&& apk add --no-cache --virtual .build-deps \
		go \
		git \
		gcc \
		libc-dev \
		libgcc \
	&& cd /go/src/github.com/jfrazelle/audit \
	&& go build -o /usr/bin/audit . \
	&& apk del .build-deps \
	&& rm -rf /go \
	&& echo "Build complete."


ENTRYPOINT [ "audit" ]
