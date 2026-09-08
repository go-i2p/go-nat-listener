module github.com/go-i2p/go-nat-listener

go 1.26.8

require (
	github.com/go-i2p/logger v0.1.70001
	github.com/huin/goupnp v1.3.0
	github.com/jackpal/go-nat-pmp v1.1.0
)

require (
	github.com/sirupsen/logrus v1.10.2 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
)

retract (
	v0.1.59999
	v0.1.5999
	v0.1.599
)
