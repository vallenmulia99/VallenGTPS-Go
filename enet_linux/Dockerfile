ARG GO_VERSION=1.22.2

FROM golang:${GO_VERSION}

RUN mkdir -p /gotops
WORKDIR /gotops
COPY . .

RUN ls -la
RUN ls -la enet
RUN git submodule update --remote --merge --init enet

# Install enet.
# Installs to: /usr/local/lib/libenet.so
RUN apt update
RUN apt install -y autoconf libtool make
RUN cd enet && \
autoreconf -vfi && \
./configure && \
make && \
make install

# Ensure we can find enet at runtime.
ENV LD_LIBRARY_PATH=/usr/local/lib
