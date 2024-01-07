FROM golang:alpine

COPY . /fserver
RUN cd /fserver & go build

EXPOSE 9090
CMD ["./fserver"]