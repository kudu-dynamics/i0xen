#!/bin/bash

apt update
apt install -y --no-install-recommends \
    ca-certificates curl gpg

# Install pre-built Zeek binaries.
#
# https://software.opensuse.org//download.html?project=security%3Azeek&package=zeek-lts
echo 'deb http://download.opensuse.org/repositories/security:/zeek/xUbuntu_18.04/ /' \
  | tee /etc/apt/sources.list.d/security:zeek.list
curl -fsSL https://download.opensuse.org/repositories/security:zeek/xUbuntu_18.04/Release.key \
  | gpg --dearmor \
  | tee /etc/apt/trusted.gpg.d/security_zeek.gpg > /dev/null

apt update
apt install -y --no-install-recommends \
    zeek-lts
