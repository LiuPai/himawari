package config

import (
	"time"
)

var (
	// HTTPRetryTimes how many times will http client retry when meet error
	HTTPRetryTimes = 1
	// HTTPTimesout how many seconds http client cancle request
        HTTPTimesout = time.Second * 10
)
