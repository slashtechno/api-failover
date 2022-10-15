FROM golang
WORKDIR /go/src/github.com/slashtechno/api-fallback
COPY . ./
RUN go install .
ENTRYPOINT ["/go/bin/api-fallback"]