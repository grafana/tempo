module github.com/soheilhy/cmux/example

go 1.11

require (
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/soheilhy/cmux v0.0.0-00010101000000-000000000000
	golang.org/x/net v0.0.0-20201202161906-c7110b5ffcbb
	google.golang.org/genproto v0.0.0-20201207150747-9ee31aac76e7 // indirect
	google.golang.org/grpc v1.27.0
)

replace github.com/soheilhy/cmux => ../
