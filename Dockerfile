FROM golang:1.21 as builder

# Create app folder
WORKDIR /app

COPY . .

# Make the binary
RUN make install

# Copy the binary from builder
# COPY --from=builder mrmintd /usr/bin/mrmintd


# COPY --from=builder mrmintd /usr/bin/mrmintd
# COPY mrmintd /usr/bin/mrmintd

# ENTRYPOINT ["mrmintd"]




# Expose necessary ports
EXPOSE 36666 36667 3092 3093 3535

# Start command
CMD ["mrmintchain"]
# CMD ["mrmintd", "start", \ 
#      "--home", "kamlesh", \
#      "--p2p.laddr", "tcp://0.0.0.0:36666", \
#      "--rpc.laddr", "tcp://0.0.0.0:36667", \
#      "--grpc.address", "0.0.0.0:3092", \
#      "--grpc-web.address", "0.0.0.0:3093", \
#      "--json-rpc.address", "0.0.0.0:3535", \
#      "--p2p.persistent_peers", "29996f0c7cc853d551e280a8162480fcd684f0b8@127.0.0.1:26656"]