module example

go 1.22

require mycache v0.0.0

require (
	github.com/golang/protobuf v1.5.4 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace mycache => ./mycache
