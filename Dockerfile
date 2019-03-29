FROM alpine:3.9
LABEL maintainer="jan.knipper@sap.com"

RUN apk --no-cache add ca-certificates
COPY concourse-ci-cleanup /concourse-ci-cleanup
USER nobody:nobody

ENTRYPOINT ["/concourse-ci-cleanup"]
CMD ["-logtostderr"]
