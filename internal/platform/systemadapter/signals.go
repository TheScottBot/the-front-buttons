package systemadapter

import (
	"os"
	"syscall"
)

// ShutdownSignals are the signals that stop the daemon cleanly, resetting the
// deck on the way out. An interrupt comes from a terminal; terminate is what the
// macOS login agent sends to stop it (TE10), and without catching it the daemon
// would end with stale images left on the keys. Windows sends neither to a
// daemon with no console, and its scheduled task stops by killing the process,
// so there the daemon is stopped by a StopRequest instead.
func ShutdownSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}
