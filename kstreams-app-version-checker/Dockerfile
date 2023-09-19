FROM alpine:3.18

ENV KUBECTL_VERSION=v1.25.10

RUN apk add curl jq --no-cache && \
    curl -LO "https://dl.k8s.io/release/$KUBECTL_VERSION/bin/linux/amd64/kubectl" && \
    curl -LO "https://dl.k8s.io/$KUBECTL_VERSION/bin/linux/amd64/kubectl.sha256" && \
    echo "$(cat kubectl.sha256)  kubectl" | sha256sum -c && \
    rm kubectl.sha256 && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/kubectl && \
    kubectl version --client

COPY run.sh /opt/hypertrace/run.sh

RUN chmod 755 /opt/hypertrace/run.sh

ENTRYPOINT ["/opt/hypertrace/run.sh"]
