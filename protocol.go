package gapless

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Connection object which handles the reading/writing and opening/closing of a connection.
// This file is a modified version from this repo: https://github.com/Mistobaan/go-apns/blob/master/protocol.go
type apnsConn struct {
	tlsconn          *tls.Conn
	tls_cfg          tls.Config
	endpoint         string
	ReadTimeout      time.Duration
	mu               sync.Mutex
	transactionId    uint32
	MAX_PAYLOAD_SIZE int
	connected        bool
}

func (client *apnsConn) connect() (err error) {
	if client.connected {
		return nil
	}

	if client.tlsconn != nil {
		client.shutdown()
	}

	conn, err := net.Dial("tcp", client.endpoint)

	if err != nil {
		return err
	}

	client.tlsconn = tls.Client(conn, &client.tls_cfg)

	err = client.tlsconn.Handshake()

	if err == nil {
		client.connected = true
	}

	return err
}

func newApnsClient(endpoint, certificate, key string) (*apnsConn, error) {
	cert, err := tls.LoadX509KeyPair(certificate, key)
	if err != nil {
		return nil, err
	}

	apnsConn := &apnsConn{
		tlsconn: nil,
		tls_cfg: tls.Config{
			InsecureSkipVerify: true,
			Certificates:       []tls.Certificate{cert},
		},
		endpoint:         endpoint,
		ReadTimeout:      150 * time.Millisecond,
		MAX_PAYLOAD_SIZE: 256,
		connected:        false,
	}

	return apnsConn, nil
}

func (client *apnsConn) shutdown() (err error) {
	err = nil
	if client.tlsconn != nil {
		err = client.tlsconn.Close()
		client.connected = false
	}
	return
}

func bwrite(w io.Writer, values ...interface{}) (err error) {
	for _, v := range values {
		err := binary.Write(w, binary.BigEndian, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func createCommandOnePacket(transactionId uint32, expiration time.Duration, token, payload []byte) ([]byte, error) {
	expirationTime := uint32(time.Now().In(time.UTC).Add(expiration).Unix())
	buffer := bytes.NewBuffer([]byte{})

	err := bwrite(buffer, uint8(1),
		transactionId,
		expirationTime,
		uint16(len(token)),
		token,
		uint16(len(payload)),
		payload)

	if err != nil {
		return nil, err
	}

	pdu := buffer.Bytes()

	return pdu, nil
}

var errText = map[uint8]string{
	0:   "No errors encountered",
	1:   "Processing Errors",
	2:   "Missing Device Token",
	3:   "Missing Topic",
	4:   "Missing Payload",
	5:   "Invalid Token Size",
	6:   "Invalid Topic Size",
	7:   "Invalid Payload Size",
	8:   "Invalid Token",
	255: "None (Unknown)",
}

// SendPayload sends push to the device (via Apple of course).
// The commands waits for a response for no more that client.ReadTimeout.
// The method uses the same connection. If the connection is closed it tries
// to reopen it at the next time.
func (client *apnsConn) SendPayload(token, payload []byte, expiration time.Duration, identity uint32) (err error) {
	if len(payload) > client.MAX_PAYLOAD_SIZE {
		return errors.New(fmt.Sprintf("The payload exceeds maximum allowed. It was: %d", len(payload)))
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	defer func() {
		if err != nil {
			client.shutdown()
		}
	}()

	// try to connect
	err = client.connect()
	if err != nil {
		return err
	}

	var pkt []byte

	pkt, err = createCommandOnePacket(identity, expiration, token, payload)
	if err != nil {
		return
	}

	_, err = client.tlsconn.Write(pkt)

	if err != nil {
		return
	}

	client.tlsconn.SetReadDeadline(time.Now().Add(client.ReadTimeout))

	readb := [6]byte{}

	n, err := client.tlsconn.Read(readb[:])

	if err != nil {
		if e2, ok := err.(net.Error); ok && e2.Timeout() {
			err = nil
			return
		} else {
			return err
		}
	}

	if n > 1 {
		var status uint8 = uint8(readb[1])

		switch status {
		case 0:
			// pass
		case 1, 2, 3, 4, 5, 6, 7, 8, 255:
			return errors.New(errText[status])
		default:
			return errors.New(fmt.Sprintf("Unknown error code %s ", hex.EncodeToString(readb[:n])))
		}
	}

	err = nil
	return
}
