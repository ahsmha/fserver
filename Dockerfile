FROM golang:alpine

COPY . /fserver
WORKDIR /fserver
RUN go mod tidy & go build

EXPOSE 9090
CMD ["./fserver"]
