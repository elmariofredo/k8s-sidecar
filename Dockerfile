FROM alpine:3.6

RUN apk --no-cache add ca-certificates tini curl bash

COPY bin/sidecar /sidecar
COPY bin/amtool /amtool
CMD chmod 755 /sidecar \
    chmod 755 /amtool

ENTRYPOINT ["/bin/bash", "-c", "/sidecar \"$@\"", "--"]