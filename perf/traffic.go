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

func mustParseTrafficType(s string) TrafficType {
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
