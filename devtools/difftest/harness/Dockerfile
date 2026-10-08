ARG IMAGE
FROM debian:trixie AS build
RUN apt-get update && apt-get install -y --no-install-recommends gcc libc6-dev && rm -rf /var/lib/apt/lists/*
COPY strcoll.c towctrans.c /src/
RUN gcc -O2 -Wall -o /strcoll /src/strcoll.c && gcc -O2 -Wall -o /towctrans /src/towctrans.c

# The harness runs on the unmodified image so that strcoll and towlower are the image's own.
FROM ${IMAGE}
COPY --from=build /strcoll /towctrans /usr/local/bin/
CMD ["sleep", "infinity"]
