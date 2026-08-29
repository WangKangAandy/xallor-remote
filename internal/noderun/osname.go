package noderun

import stdruntime "runtime"

func osName() string { return stdruntime.GOOS }
