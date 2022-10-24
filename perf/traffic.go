package main

type TrafficType string

const (
	EdgeTraffic        TrafficType = "edge"
	HTTPTraffic        TrafficType = "http"
	PassthroughTraffic TrafficType = "passthrough"
	ReencryptTraffic   TrafficType = "reencrypt"
)

var AllTrafficTypes = [...]TrafficType{
	EdgeTraffic,
	HTTPTraffic,
	PassthroughTraffic,
	ReencryptTraffic,
}

func (t TrafficType) Scheme() string {
	switch t {
	case HTTPTraffic:
		return "http"
	default:
		return "https"
	}
}

func (t TrafficType) Port() int64 {
	switch t {
	case HTTPTraffic:
		return 8080
	default:
		return 8443
	}
}

func ParseTrafficType(s string) TrafficType {
	switch s {
	case "http":
		return HTTPTraffic
	case "edge":
		return EdgeTraffic
	case "reencrypt":
		return ReencryptTraffic
	case "passthrough":
		return PassthroughTraffic
	}
	panic("unknown taffic type" + s)
}
