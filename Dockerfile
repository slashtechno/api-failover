FROM golang:1.19.2-bullseye
# WORKDIR /go/src/github.com/slashtechno/api-fallback
# COPY . ./
# RUN go install
RUN go install github.com/slashtechno/api-fallback@latest 
ENTRYPOINT ["/go/bin/api-fallback"]