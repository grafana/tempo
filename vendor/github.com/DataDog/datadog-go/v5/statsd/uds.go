// +build !windows

package statsd

import (
	"net"
	"sync"
	"time"
)

// udsWriter is an internal class wrapping around management of UDS connection
type udsWriter struct {
	// Address to send metrics to, needed to allow reconnection on error
	addr net.Addr
	// Established connection object, or nil if not connected yet
	conn net.Conn
	// write timeout
	writeTimeout time.Duration
	sync.RWMutex // used to lock conn / writer can replace it
}

// newUDSWriter returns a pointer to a new udsWriter given a socket file path as addr.
func newUDSWriter(addr string, writeTimeout time.Duration) (*udsWriter, error) {
	udsAddr, err := net.ResolveUnixAddr("unixgram", addr)
	if err != nil {
		return nil, err
	}
	// Defer connection to first Write
	writer := &udsWriter{addr: udsAddr, conn: nil, writeTimeout: writeTimeout}
	return writer, nil
}

// Write data to the UDS connection with write timeout and minimal error handling:
// create the connection if nil, and destroy it if the statsd server has disconnected
func (w *udsWriter) Write(data []byte) (int, error) {
	conn, err := w.ensureConnection()
	if err != nil {
		return 0, err
	}

	conn.SetWriteDeadline(time.Now().Add(w.writeTimeout))
	n, e := conn.Write(data)

	if err, isNetworkErr := e.(net.Error); err != nil && (!isNetworkErr || !err.Temporary()) {
		// Statsd server disconnected, retry connecting at next packet
		w.unsetConnection()
		return 0, e
	}
	return n, e
}

func (w *udsWriter) Close() error {
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

func (w *udsWriter) ensureConnection() (net.Conn, error) {
	// Check if we've already got a socket we can use
	w.RLock()
	currentConn := w.conn
	w.RUnlock()

	if currentConn != nil {
		return currentConn, nil
	}

	// Looks like we might need to connect - try again with write locking.
	w.Lock()
	defer w.Unlock()
	if w.conn != nil {
		return w.conn, nil
	}

	newConn, err := net.Dial(w.addr.Network(), w.addr.String())
	if err != nil {
		return nil, err
	}
	w.conn = newConn
	return newConn, nil
}

func (w *udsWriter) unsetConnection() {
	w.Lock()
	defer w.Unlock()
	w.conn = nil
}
