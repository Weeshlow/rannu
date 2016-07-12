FROM scratch

ADD ./bin/worker /
ADD ./data /data

ENTRYPOINT ["/worker"]

EXPOSE 8000
