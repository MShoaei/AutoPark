FROM alpine
RUN mkdir -p /go/bin
WORKDIR /go/bin
ADD AutoPark .
#RUN go build .
RUN pwd & ls
CMD ["./AutoPark"]