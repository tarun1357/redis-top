package redis

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

// Client is a lightweight, zero-dependency RESP (Redis Serialization Protocol) client
// supporting non-blocking SCAN and pipelined MEMORY USAGE + TTL inspection.
type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

// Dial connects to a Redis instance, authenticates if password is set, and selects the DB.
func Dial(addr string, password string, db int, timeout time.Duration) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	c := &Client{
		conn:   conn,
		reader: bufio.NewReaderSize(conn, 64*1024),
		writer: bufio.NewWriterSize(conn, 64*1024),
	}

	if password != "" {
		if err := c.writeCommand("AUTH", password); err != nil {
			conn.Close()
			return nil, err
		}
		if err := c.writer.Flush(); err != nil {
			conn.Close()
			return nil, err
		}
		if _, err := c.readReply(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("AUTH failed: %w", err)
		}
	}

	if db > 0 {
		if err := c.writeCommand("SELECT", strconv.Itoa(db)); err != nil {
			conn.Close()
			return nil, err
		}
		if err := c.writer.Flush(); err != nil {
			conn.Close()
			return nil, err
		}
		if _, err := c.readReply(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("SELECT %d failed: %w", db, err)
		}
	}

	return c, nil
}

// Close closes the underlying TCP socket.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Scan executes a single non-blocking SCAN iteration.
func (c *Client) Scan(cursor string, match string, count int) (string, []string, error) {
	args := []string{"SCAN", cursor}
	if match != "" && match != "*" {
		args = append(args, "MATCH", match)
	}
	if count > 0 {
		args = append(args, "COUNT", strconv.Itoa(count))
	}

	if err := c.writeCommand(args...); err != nil {
		return "0", nil, err
	}
	if err := c.writer.Flush(); err != nil {
		return "0", nil, err
	}

	reply, err := c.readReply()
	if err != nil {
		return "0", nil, err
	}

	arr, ok := reply.([]interface{})
	if !ok || len(arr) != 2 {
		return "0", nil, errors.New("unexpected SCAN response structure")
	}

	nextCursor, _ := arr[0].(string)
	rawKeys, _ := arr[1].([]interface{})
	keys := make([]string, 0, len(rawKeys))
	for _, rk := range rawKeys {
		if k, ok := rk.(string); ok {
			keys = append(keys, k)
		}
	}
	return nextCursor, keys, nil
}

// InspectBatch pipelines MEMORY USAGE and TTL commands for a batch of keys in a single round-trip.
func (c *Client) InspectBatch(keys []string) ([]int64, []int64, error) {
	if len(keys) == 0 {
		return nil, nil, nil
	}

	for _, k := range keys {
		if err := c.writeCommand("MEMORY", "USAGE", k); err != nil {
			return nil, nil, err
		}
		if err := c.writeCommand("TTL", k); err != nil {
			return nil, nil, err
		}
	}

	if err := c.writer.Flush(); err != nil {
		return nil, nil, err
	}

	memList := make([]int64, len(keys))
	ttlList := make([]int64, len(keys))

	for i := range keys {
		memReply, err := c.readReply()
		if err != nil {
			return nil, nil, err
		}
		if v, ok := memReply.(int64); ok {
			memList[i] = v
		}

		ttlReply, err := c.readReply()
		if err != nil {
			return nil, nil, err
		}
		if v, ok := ttlReply.(int64); ok {
			ttlList[i] = v
		}
	}

	return memList, ttlList, nil
}

func (c *Client) writeCommand(args ...string) error {
	if _, err := fmt.Fprintf(c.writer, "*%d
", len(args)); err != nil {
		return err
	}
	for _, arg := range args {
		if _, err := fmt.Fprintf(c.writer, "$%d
%s
", len(arg), arg); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) readReply() (interface{}, error) {
	line, err := c.reader.ReadString('
')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSuffix(strings.TrimSuffix(line, "
"), "")
	if len(line) == 0 {
		return nil, errors.New("empty RESP reply line")
	}

	prefix := line[0]
	payload := line[1:]

	switch prefix {
	case '+': // Simple String
		return payload, nil
	case '-': // Error
		return nil, errors.New(payload)
	case ':': // Integer
		return strconv.ParseInt(payload, 10, 64)
	case '$': // Bulk String
		length, err := strconv.Atoi(payload)
		if err != nil {
			return nil, err
		}
		if length < 0 {
			return nil, nil // Null bulk string
		}
		buf := make([]byte, length+2)
		if _, err := io.ReadFull(c.reader, buf); err != nil {
			return nil, err
		}
		return string(buf[:length]), nil
	case '*': // Array
		count, err := strconv.Atoi(payload)
		if err != nil {
			return nil, err
		}
		if count < 0 {
			return nil, nil
		}
		items := make([]interface{}, count)
		for i := 0; i < count; i++ {
			item, err := c.readReply()
			if err != nil {
				return nil, err
			}
			items[i] = item
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unknown RESP prefix byte: %q", prefix)
	}
}
