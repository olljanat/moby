package client

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/moby/moby/api/types/versions"
	"github.com/pkg/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// postHijacked sends a POST request and hijacks the connection.
func (cli *Client) postHijacked(ctx context.Context, path string, query url.Values, body interface{}, headers map[string][]string) (HijackedResponse, error) {
	jsonBody, err := jsonEncode(body)
	if err != nil {
		return HijackedResponse{}, err
	}
	req, err := cli.buildRequest(ctx, http.MethodPost, cli.getAPIPath(ctx, path, query), jsonBody, headers)
	if err != nil {
		return HijackedResponse{}, err
	}
	conn, mediaType, err := setupHijackConn(cli.dialer(), req, "tcp")
	if err != nil {
		return HijackedResponse{}, err
	}

	if versions.LessThan(cli.ClientVersion(), "1.42") {
		// Prior to 1.42, Content-Type is always set to raw-stream and not relevant
		mediaType = ""
	}

	return NewHijackedResponse(conn, mediaType), nil
}

// DialHijack returns a hijacked connection with negotiated protocol proto.
func (cli *Client) DialHijack(ctx context.Context, url, proto string, meta map[string][]string) (net.Conn, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	req = cli.addHeaders(req, meta)

	conn, _, err := setupHijackConn(cli.Dialer(), req, proto)
	return conn, err
}

func setupHijackConn(dialer func(context.Context) (net.Conn, error), req *http.Request, proto string) (_ net.Conn, _ string, retErr error) {
	ctx := req.Context()
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", proto)

	conn, err := dialer(ctx)
	if err != nil {
		return nil, "", errors.Wrap(err, "cannot connect to the Docker daemon. Is 'docker daemon' running on this host?")
	}
	defer func() {
		if retErr != nil {
			conn.Close()
		}
	}()

	// When we set up a TCP connection for hijack, there could be long periods
	// of inactivity (a long running command with no output) that in certain
	// network setups may cause ECONNTIMEOUT, leaving the client in an unknown
	// state. Setting TCP KeepAlive on the socket connection will prohibit
	// ECONNTIMEOUT unless the socket connection truly is broken
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetKeepAlive(true)
		_ = tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	hc := &hijackedConn{conn, bufio.NewReader(conn)}

	// Server hijacks the connection, error 'connection closed' expected
	resp, err := otelhttp.NewTransport(hc).RoundTrip(req)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		_ = resp.Body.Close()
		return nil, "", fmt.Errorf("unable to upgrade to %s, received %d", proto, resp.StatusCode)
	}

	if hc.r.Buffered() > 0 {
		// If there is buffered content, wrap the connection.  We return an
		// object that implements CloseWrite if the underlying connection
		// implements it.
		if _, ok := hc.Conn.(CloseWriter); ok {
			conn = &hijackedConnCloseWriter{hc}
		} else {
			conn = hc
		}
	} else {
		hc.r.Reset(nil)
	}

	return conn, resp.Header.Get("Content-Type"), nil
}

// hijackedConn wraps a net.Conn and is returned by setupHijackConn in the case
// that a) there was already buffered data in the http layer when Hijack() was
// called, and b) the underlying net.Conn does *not* implement CloseWrite().
// hijackedConn does not implement CloseWrite() either.
type hijackedConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *hijackedConn) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := req.Write(c.Conn); err != nil {
		return nil, err
	}
	return http.ReadResponse(c.r, req)
}

func (c *hijackedConn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

// hijackedConnCloseWriter is a hijackedConn which additionally implements
// CloseWrite().  It is returned by setupHijackConn in the case that a) there
// was already buffered data in the http layer when Hijack() was called, and b)
// the underlying net.Conn *does* implement CloseWrite().
type hijackedConnCloseWriter struct {
	*hijackedConn
}

var _ CloseWriter = &hijackedConnCloseWriter{}

func (c *hijackedConnCloseWriter) CloseWrite() error {
	conn := c.Conn.(CloseWriter)
	return conn.CloseWrite()
}

// NewHijackedResponse initializes a [HijackedResponse] type.
func NewHijackedResponse(conn net.Conn, mediaType string) HijackedResponse {
	return HijackedResponse{Conn: conn, Reader: bufio.NewReader(conn), mediaType: mediaType}
}

// HijackedResponse holds connection information for a hijacked request.
type HijackedResponse struct {
	mediaType string
	Conn      net.Conn
	Reader    *bufio.Reader
}

// Close closes the hijacked connection and reader.
func (h *HijackedResponse) Close() {
	h.Conn.Close()
}

// MediaType let client know if HijackedResponse hold a raw or multiplexed stream.
// returns false if HTTP Content-Type is not relevant, and container must be inspected
func (h *HijackedResponse) MediaType() (string, bool) {
	if h.mediaType == "" {
		return "", false
	}
	return h.mediaType, true
}

// CloseWriter is an interface that implements structs
// that close input streams to prevent from writing.
type CloseWriter interface {
	CloseWrite() error
}

// CloseWrite closes a readWriter for writing.
func (h *HijackedResponse) CloseWrite() error {
	if conn, ok := h.Conn.(CloseWriter); ok {
		return conn.CloseWrite()
	}
	return nil
}
