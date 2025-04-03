FROM debian:latest

RUN apt-get update && apt-get install -y \
    glusterfs-client \
    fuse \
    && apt-get clean

RUN mkdir -p /mounted

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]