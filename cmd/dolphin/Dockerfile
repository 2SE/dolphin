FROM golang:1.12.4

WORKDIR /app


#ENV GOPROXY=https://goproxy.io
#ENV GO111MODULE=on

#COPY . .

#RUN go mod download

#RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o myapp

#rm -rf myapp


COPY myapp  /app/

EXPOSE 8080
EXPOSE 41050
EXPOSE 40051
EXPOSE 9527
EXPOSE 9528

ENTRYPOINT ["./myapp","-c","config.toml"]


