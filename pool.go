package gapless

// Setup the connection pool.
type connectionPoolWrapper struct {
    size int
    conn chan *ApnsConn
}

var ConnPool = &connectionPoolWrapper{}

func (p *connectionPoolWrapper) InitPool(size int, server, cert, key string) error {

    // Create a buffered channel allowing size senders
    p.conn = make(chan *ApnsConn, size)
    for x := 0; x < size; x++ {
        conn, err := NewApnsClient(server, cert, key)
        if err != nil {
            return err
        }

        stdout.Printf("Starting apns connection: #%d", x)
        p.conn <- conn
    }
    p.size = size
    return nil
}

func (p *connectionPoolWrapper) GetConn() *ApnsConn {
    return <-p.conn
}

func (p *connectionPoolWrapper) ReleaseConn(conn *ApnsConn) {
    p.conn <- conn
}

func (p *connectionPoolWrapper) ShutdownConns() {
    for x := 0; x < p.size; x++ {
        tmp := <-p.conn
        tmp.shutdown()
    }

    close(p.conn)
}
