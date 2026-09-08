FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/valueops ./cmd/valueops

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/valueops /valueops
COPY config/rate-cards.json /config/rate-cards.json
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/valueops"]
