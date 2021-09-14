package router

const (
	// Tester Headers Names
	maxIdleConnPerHostHeader = "T-Max-Idle-Conn-Host"
	disableCompressionHeader = "T-Disable-Compress"
	disableKeepAliveHeader   = "T-Disable-Keep-Alive"
	reqTimeoutHeader         = "T-Req-Timeout"
	reqMethodHeader          = "T-Method"
	reqAcceptHeader          = "T-Accept"
	reqUserAgentHeader       = "T-User-Agent"

	// Query Params Names
	maxIdleConnPerHostParam = "tmaxidleconnhost"
	disableCompressionParam = "tdisablecompress"
	disableKeepAliveParam   = "tdisablekeepalive"
	reqTimeoutParam         = "treqtimeout"
	reqMethodParam          = "tmethod"
)
