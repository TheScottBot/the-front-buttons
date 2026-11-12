package systemadapter

import (
	"os"
	"syscall"
)

// ShutdownSignals are the signals that stop the daemon cleanly, resetting the
// deck on the way out. An interrupt comes from a terminal; terminate is what the
// macOS login agent and the Windows scheduled task send to stop it (TE10), and
// without catching it the daemon would end with stale images left on the keys.
func ShutdownSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}
