FROM alpine:3.19

ARG TARGETARCH

COPY bin/sub-store-linux-${TARGETARCH} /sub-store

RUN chmod +x /sub-store

EXPOSE 8080

ENV PATH=""

CMD /sub-store --port=8080 --dir=/data --path=${PATH}
