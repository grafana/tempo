FROM alpine:3.15 as certs
RUN apk --update add ca-certificates
ARG TARGETARCH
COPY bin/linux/tempo-${TARGETARCH} /tempo

# /var/tempo is a default directory used by Tempo components.
# At runtime the Tempo process creates this dectory and it needs write access to it.
# On OpenShift the tempo user is not root (but in root group) and it needs access to the directory.
RUN mkdir -p /var/tempo
RUN chgrp -R 0 /var/tempo && chmod -R g+rwX /var/tempo

ENTRYPOINT ["/tempo"]
