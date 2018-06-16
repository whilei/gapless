package gapless

// Setup the connection pool.
type connectionPoolWrapper struct {
	size int
	conn chan *apnsConn
}

// Holds individual connections to Apple's push servers.
var connPool = &connectionPoolWrapper{}

// InitPool populates the connection pool with the correct number of connections.
func (p *connectionPoolWrapper) InitPool(size int, server, cert, key string) error {
	p.conn = make(chan *apnsConn, size)
	for x := 0; x < size; x++ {
		conn, err := newApnsClient(server, cert, key)
		if err != nil {
			return err
		}

		stdout.Printf("Starting apns connection: #%d", x)
		p.conn <- conn
	}
	p.size = size
	return nil
}

// Grab a connection from the pool.
// If the pool has no available connections, this will block until one becomes available.
func (p *connectionPoolWrapper) GetConn() *apnsConn {
	return <-p.conn
}

// Returns the connection back into the pool for reuse.
func (p *connectionPoolWrapper) ReleaseConn(conn *apnsConn) {
	p.conn <- conn
}

// Gracefully close all the connections.
func (p *connectionPoolWrapper) ShutdownConns() {
	for x := 0; x < p.size; x++ {
		tmp := <-p.conn
		tmp.shutdown()
	}

	close(p.conn)
}
