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

# RUN chmod +x entrypoint.sh startnode.sh
# ENTRYPOINT ["./entrypoint.sh"]


# Expose necessary ports
# EXPOSE 36666 36667 3092 3093 3535

