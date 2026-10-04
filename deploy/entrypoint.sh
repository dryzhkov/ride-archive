#!/bin/sh
set -eu
# The mounted volume starts root-owned; the API itself runs as archive.
mkdir -p /data
chown archive:archive /data
chmod 700 /data
exec su-exec archive /usr/local/bin/ride-archive
