package ws

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/gorilla/websocket"
)

// ---------------------- CLIENT ----------------------

// Client defines a websocket client, needed to connect to a websocket server.
// The offered API are of asynchronous nature, and each incoming message is handled using callbacks.
//
// To create a new ws client, use:
//
//	client := NewClient()
//
// If you need a TLS ws client instead, use:
//
//	certPool, err := x509.SystemCertPool()
//	if err != nil {
//		log.Fatal(err)
//	}
//	// You may add more trusted certificates to the pool before creating the TLSClientConfig
//	client := NewTLSClient(&tls.Config{
//		RootCAs: certPool,
//	})
//
// To add additional dial options, use:
//
//	client.AddOption(func(*websocket.Dialer) {
//		// Your option ...
//	})
//
// To add basic HTTP authentication, use:
//
//	client.SetBasicAuth("username","password")
//
// If you need to set a specific timeout configuration, refer to the SetTimeoutConfig method.
//
// Using Start and Stop you can respectively open/close a websocket to a websocket server.
//
// To receive incoming messages, you will need to set your own handler using SetMessageHandler.
// To write data on the open socket, simply call the Write function.
type Client interface {
	// Starts the client and attempts to connect to the server on a specified URL.
	// If the connection fails, an error is returned.
	//
	// For example:
	//	err := client.Start("ws://localhost:8887/ws/1234")
	//
	// The function returns immediately, after the connection has been established.
	// Incoming messages are passed automatically to the callback function, so no explicit read operation is required.
	//
	// To stop a running client, call the Stop function.
	Start(url string) error
	// Starts the client and attempts to connect to the server on a specified URL.
	// If the connection fails, it keeps retrying with Backoff strategy from TimeoutConfig.
	//
	// For example:
	//	client.StartWithRetries("ws://localhost:8887/ws/1234")
	//
	// The function returns only when the connection has been established.
	// Incoming messages are passed automatically to the callback function, so no explicit read operation is required.
	//
	// To stop a running client, call the Stop function.
	StartWithRetries(url string)
	// Stop closes the output of the websocket Channel, effectively closing the connection to the server with a normal closure.
	Stop()
	// Errors returns a channel for error messages. If it doesn't exist it es created.
	// The channel is closed by the client when stopped.
	//
	// It is recommended to invoke this function before starting a client.
	// Creating the error channel while the client is running may lead to unexpected behavior.
	Errors() <-chan error
	// Sets a callback function for all incoming messages.
	SetMessageHandler(handler func(data []byte) error)
	// Set custom timeout configuration parameters. If not passed, a default ClientTimeoutConfig struct will be used.
	//
	// This function must be called before connecting to the server, otherwise it may lead to unexpected behavior.
	SetTimeoutConfig(config ClientTimeoutConfig)
	// Sets a callback function for receiving notifications about an unexpected disconnection from the server.
	// The callback is invoked even if the automatic reconnection mechanism is active.
	//
	// If the client was stopped using the Stop function, the callback will NOT be invoked.
	SetDisconnectedHandler(handler func(err error))
	// Sets a callback function for receiving notifications whenever the connection to the server is re-established.
	// Connections are re-established automatically thanks to the auto-reconnection mechanism.
	//
	// If set, the DisconnectedHandler will always be invoked before the Reconnected callback is invoked.
	SetReconnectedHandler(handler func())
	// IsConnected Returns information about the current connection status.
	// If the client is currently attempting to auto-reconnect to the server, the function returns false.
	IsConnected() bool
	// Sends a message to the server over the websocket.
	//
	// The data is queued and will be sent asynchronously in the background.
	Write(data []byte) error
	// Adds a websocket option to the client.
	AddOption(option interface{})
	// SetRequestedSubProtocol will negotiate the specified sub-protocol during the websocket handshake.
	// Internally this creates a dialer option and invokes the AddOption method on the client.
	//
	// Duplicates generated by invoking this method multiple times will be ignored.
	SetRequestedSubProtocol(subProto string)
	// SetBasicAuth adds basic authentication credentials, to use when connecting to the server.
	// The credentials are automatically encoded in base64.
	SetBasicAuth(username string, password string)
	// SetHeaderValue sets a value on the HTTP header sent when opening a websocket connection to the server.
	//
	// The function overwrites previous header fields with the same key.
	SetHeaderValue(key string, value string)
}

// client is the default implementation of a Websocket client.
//
// Use the NewClient or NewTLSClient functions to create a new client.
type client struct {
	webSocket      *webSocket
	url            url.URL
	messageHandler func(data []byte) error
	dialOptions    []func(*websocket.Dialer)
	header         http.Header
	timeoutConfig  ClientTimeoutConfig
	onDisconnected func(err error)
	onReconnected  func()
	errC           chan error
	reconnectC     chan struct{} // used for signaling, that a reconnection attempt should be interrupted
}

// Creates a new simple websocket client (the channel is not secured).
//
// Additional options may be added using the AddOption function.
//
// Basic authentication can be set using the SetBasicAuth function.
//
// By default, the client will not neogtiate any subprotocol. This value needs to be set via the
// respective SetRequestedSubProtocol method.
func NewClient() Client {
	return &client{
		dialOptions:   []func(*websocket.Dialer){},
		timeoutConfig: NewClientTimeoutConfig(),
		reconnectC:    make(chan struct{}, 1),
		header:        http.Header{},
	}
}

// NewTLSClient creates a new secure websocket client. If supported by the server, the websocket channel will use TLS.
//
// Additional options may be added using the AddOption function.
// Basic authentication can be set using the SetBasicAuth function.
//
// To set a client certificate, you may do:
//
//	certificate, _ := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
//	clientCertificates := []tls.Certificate{certificate}
//	client := ws.NewTLSClient(&tls.Config{
//		RootCAs:      certPool,
//		Certificates: clientCertificates,
//	})
//
// You can set any other TLS option within the same constructor as well.
// For example, if you wish to test connecting to a server having a
// self-signed certificate (do not use in production!), pass:
//
//	InsecureSkipVerify: true
func NewTLSClient(tlsConfig *tls.Config) Client {
	c := &client{
		dialOptions:   []func(*websocket.Dialer){},
		timeoutConfig: NewClientTimeoutConfig(),
		reconnectC:    make(chan struct{}, 1),
		header:        http.Header{},
	}
	c.dialOptions = append(c.dialOptions, func(dialer *websocket.Dialer) {
		dialer.TLSClientConfig = tlsConfig
	})
	return c
}

func (c *client) SetMessageHandler(handler func(data []byte) error) {
	c.messageHandler = handler
}

func (c *client) SetTimeoutConfig(config ClientTimeoutConfig) {
	c.timeoutConfig = config
}

func (c *client) SetDisconnectedHandler(handler func(err error)) {
	c.onDisconnected = handler
}

func (c *client) SetReconnectedHandler(handler func()) {
	c.onReconnected = handler
}

func (c *client) AddOption(option interface{}) {
	dialOption, ok := option.(func(*websocket.Dialer))
	if ok {
		c.dialOptions = append(c.dialOptions, dialOption)
	}
}

func (c *client) SetRequestedSubProtocol(subProto string) {
	opt := func(dialer *websocket.Dialer) {
		alreadyExists := false
		for _, proto := range dialer.Subprotocols {
			if proto == subProto {
				alreadyExists = true
				break
			}
		}
		if !alreadyExists {
			dialer.Subprotocols = append(dialer.Subprotocols, subProto)
		}
	}
	c.AddOption(opt)
}

func (c *client) SetBasicAuth(username string, password string) {
	c.header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
}

func (c *client) SetHeaderValue(key string, value string) {
	c.header.Set(key, value)
}

func (c *client) getReadTimeout() time.Time {
	if c.timeoutConfig.PongWait == 0 {
		return time.Time{}
	}
	return time.Now().Add(c.timeoutConfig.PongWait)
}

func (c *client) handleReconnection() {
	log.Info("started automatic reconnection handler")
	delay := c.timeoutConfig.RetryBackOffWaitMinimum + time.Duration(rand.Intn(c.timeoutConfig.RetryBackOffRandomRange+1))*time.Second
	reconnectionAttempts := 1
	for {
		// Wait before reconnecting
		select {
		case <-time.After(delay):
		case <-c.reconnectC:
			log.Info("automatic reconnection aborted")
			return
		}

		log.Info("reconnecting... attempt", reconnectionAttempts)
		err := c.Start(c.url.String())
		if err == nil {
			// Re-connection was successful
			log.Info("reconnected successfully to server")
			if c.onReconnected != nil {
				c.onReconnected()
			}
			return
		}
		c.error(fmt.Errorf("reconnection failed: %w", err))

		if reconnectionAttempts < c.timeoutConfig.RetryBackOffRepeatTimes {
			// Re-connection failed, double the delay
			delay *= 2
			delay += time.Duration(rand.Intn(c.timeoutConfig.RetryBackOffRandomRange+1)) * time.Second
		}
		reconnectionAttempts += 1
	}
}

func (c *client) IsConnected() bool {
	if c.webSocket == nil {
		return false
	}
	return c.webSocket.IsConnected()
}

func (c *client) Write(data []byte) error {
	if !c.IsConnected() {
		return fmt.Errorf("client is currently not connected, cannot send data")
	}
	log.Debugf("queuing data for server")
	return c.webSocket.Write(data)
}

func (c *client) StartWithRetries(urlStr string) {
	err := c.Start(urlStr)
	if err != nil {
		log.Info("Connection error:", err)
		c.handleReconnection()
	}
}

func (c *client) Start(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}
	c.url = *u
	if c.reconnectC == nil {
		c.reconnectC = make(chan struct{}, 1)
	}

	dialer := websocket.Dialer{
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		HandshakeTimeout: c.timeoutConfig.HandshakeTimeout,
		Subprotocols:     []string{},
	}
	for _, option := range c.dialOptions {
		option(&dialer)
	}
	// Connect
	log.Info("connecting to server")
	ws, resp, err := dialer.Dial(urlStr, c.header)
	if err != nil {
		if resp != nil {
			httpError := HttpConnectionError{Message: err.Error(), HttpStatus: resp.Status, HttpCode: resp.StatusCode}
			// Parse http response details
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if body != nil {
				httpError.Details = string(body)
			}
			err = httpError
		}
		return err
	}

	// The id of the charge point is the final path element
	id := path.Base(u.Path)

	// Create web socket, state is automatically set to connected
	c.webSocket = newWebSocket(
		id,
		ws,
		resp.TLS,
		NewDefaultWebSocketConfig(
			c.timeoutConfig.WriteWait,
			0,
			c.timeoutConfig.PingPeriod,
			c.timeoutConfig.PongWait,
		),
		c.handleMessage,
		c.handleDisconnect,
		func(_ Channel, err error) {
			c.error(err)
		},
	)
	log.Infof("connected to server as %s", id)
	// Start reader and write routine
	c.webSocket.run()
	return nil
}

func (c *client) Stop() {
	log.Infof("closing connection to server")
	if c.IsConnected() {
		// Attempt to gracefully shut down the connection
		err := c.webSocket.Close(websocket.CloseError{Code: websocket.CloseNormalClosure, Text: ""})
		if err != nil {
			c.error(err)
		}
	}
	// Notify reconnection goroutine to stop (if any)
	select {
	case <-c.reconnectC:
		// Already closed, ignore
		break
	default:
		// Channel is open, signal reconnection to stop
		c.reconnectC <- struct{}{}
	}
	// Close error channel if any
	select {
	case <-c.errC:
		// Already closed, ignore
		break
	default:
		// Channel is open, close it
		if c.errC != nil {
			close(c.errC)
		}
	}
	// Connection will close asynchronously and invoke the onDisconnected handler
}

func (c *client) Errors() <-chan error {
	if c.errC == nil {
		c.errC = make(chan error, 1)
	}
	return c.errC
}

// --------- Internal callbacks webSocket -> client ---------
func (c *client) handleMessage(_ Channel, data []byte) error {
	if c.messageHandler != nil {
		return c.messageHandler(data)
	}
	return fmt.Errorf("no message handler set")
}

func (c *client) handleDisconnect(_ Channel, err error) {
	if c.onDisconnected != nil {
		// Notify upper layer of disconnect
		c.onDisconnected(err)
	}
	if err != nil {
		// Disconnect was forced, do reconnect
		c.handleReconnection()
	}
}

func (c *client) error(err error) {
	log.Error(err)
	if c.errC != nil {
		c.errC <- err
	}
}
