ARG IMAGE
FROM debian:trixie AS build
RUN apt-get update && apt-get install -y --no-install-recommends gcc libc6-dev && rm -rf /var/lib/apt/lists/*
COPY strcoll.c /src/strcoll.c
RUN gcc -O2 -Wall -o /strcoll /src/strcoll.c

# The harness runs on the unmodified image so that strcoll is the image's own.
FROM ${IMAGE}
COPY --from=build /strcoll /usr/local/bin/strcoll
CMD ["sleep", "infinity"]
