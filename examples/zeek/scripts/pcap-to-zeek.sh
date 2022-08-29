#!/bin/bash

set -e

# Download and unpack the input PCAP archives.

curl \
  -o /tmp/output.pcap.gz \
  "${NOMAD_META_download_url}"

gunzip \
  /tmp/output.pcap.gz

# Process the PCAPs with Zeek.

/opt/zeek/bin/zeek \
  -Cr \
  /tmp/*.pcap

# DEV: For debugging purposes, cat the DNS log in question.

cat \
  /tmp/*dns.log

gzip \
  /tmp/*dns.log

# Upload the DNS log.

# s3cli upload \
#   dns.log.gz \
#   "${NOMAD_META_s3_output}.dns.log.gz"
