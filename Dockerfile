FROM golang:1.14.4 as builder
WORKDIR /go/src/crd-client

COPY ./main.go .
COPY ./go.mod ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o client .

# Create the final image that will run the webhook server for FleetAutoscaler webhook policy
FROM alpine:3.11
RUN adduser -D -u 1000 client

COPY --from=builder /go/src/crd-client \
                    /home/client

# port to listen
# host for running launcher
# mongo ip (mongo db for keep subsymbolic data in Unreal)
# mongo
ENV PORT=9090 \                  
    HOST=192.168.102.21 \
    MONGO_IP=192.168.102.21 \
    MONGO_PORT=17017 \
    IMAGE_REPO=xiaojunll 

RUN chown -R client /home/client && \
    chmod o+x /home/client/client

USER 1000
ENTRYPOINT /home/client/client