FROM scratch
ENV PATH=/bin

COPY ok /bin/

WORKDIR /

ENTRYPOINT ["/bin/ok"]