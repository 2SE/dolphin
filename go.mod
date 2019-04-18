module github.com/2se/dolphin

go 1.12

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/gobwas/httphead v0.0.0-20180130184737-2c6c146eadee // indirect
	github.com/gobwas/pool v0.2.0 // indirect
	github.com/gobwas/ws v1.0.0
	github.com/golang/protobuf v1.3.2-0.20190409050943-e91709a02e0e
	github.com/google/uuid v1.1.1
	github.com/magiconair/properties v1.8.0
	github.com/segmentio/kafka-go v0.2.2 // indirect
	github.com/sirupsen/logrus v1.4.1
	golang.org/x/crypto v0.0.0-20190411191339-88737f569e3a
	golang.org/x/sys v0.0.0-20190411185658-b44545bcd369 // indirect
	google.golang.org/grpc v1.20.0
	gopkg.in/urfave/cli.v1 v1.20.0
)

replace golang.org/x/crypto v0.0.0-20190308221718-c2843e01d9a2 => github.com/golang/crypto v0.0.0-20190308221718-c2843e01d9a2

replace google.golang.org/grpc v1.20.0 => github.com/grpc/grpc-go v1.20.0

replace golang.org/x/sys v0.0.0-20190215142949-d0b11bdaac8a => github.com/golang/sys v0.0.0-20190215142949-d0b11bdaac8a

replace golang.org/x/lint v0.0.0-20190313153728-d0100b6bd8b3 => github.com/golang/lint v0.0.0-20190313153728-d0100b6bd8b3

replace golang.org/x/sync v0.0.0-20180314180146-1d60e4601c6f => github.com/golang/sync v0.0.0-20180314180146-1d60e4601c6f

replace golang.org/x/oauth2 v0.0.0-20180821212333-d2e6202438be => github.com/golang/oauth2 v0.0.0-20180821212333-d2e6202438be

replace golang.org/x/tools v0.0.0-20190311212946-11955173bddd => github.com/golang/tools v0.0.0-20190311212946-11955173bddd

replace golang.org/x/net v0.0.0-20190311183353-d8887717615a => github.com/golang/net v0.0.0-20190311183353-d8887717615a

replace cloud.google.com/go v0.26.0 => github.com/googleapis/google-cloud-go v0.26.0

replace golang.org/x/sys v0.0.0-20180905080454-ebe1bf3edb33 => github.com/golang/sys v0.0.0-20180905080454-ebe1bf3edb33

replace golang.org/x/text v0.3.0 => github.com/golang/text v0.3.0
