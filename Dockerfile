FROM public.ecr.aws/docker/library/golang:1.17 as build-env

WORKDIR /go/src/app

COPY go.* ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /go/bin/shipper ./cmd/shipper

FROM public.ecr.aws/docker/library/debian:bullseye-slim

COPY --from=build-env /go/bin/shipper /usr/local/bin
CMD ["shipper"]
