FROM debian:stable-slim

ENV PORT=8991

COPY goserver /bin/goserver

CMD ["/bin/goserver"]   