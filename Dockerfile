FROM golang:1.19.2-bullseye
# WORKDIR /go/src/github.com/slashtechno/api-failover
# COPY . ./
# RUN go install
RUN go install github.com/slashtechno/api-failover@latest 
ENTRYPOINT ["/go/bin/api-failover"]