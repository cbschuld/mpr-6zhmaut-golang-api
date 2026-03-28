package serial

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"sync"

	goserial "go.bug.st/serial"
)

// Port wraps a serial port connection with baud rate management.
type Port struct {
	mu       sync.Mutex
	device   string
	baudRate int
	port     goserial.Port
	reader   *bufio.Reader
	logger   *slog.Logger
}

// NewPort creates a new serial port wrapper (does not open it).
func NewPort(device string, logger *slog.Logger) *Port {
	return &Port{
		device: device,
		logger: logger,
	}
}

// Open opens the serial port at the specified baud rate.
func (p *Port) Open(baudRate int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.port != nil {
		p.port.Close()
		p.port = nil
	}

	mode := &goserial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   goserial.NoParity,
		StopBits: goserial.OneStopBit,
	}

	port, err := goserial.Open(p.device, mode)
	if err != nil {
		return fmt.Errorf("open serial %s at %d: %w", p.device, baudRate, err)
	}

	p.port = port
	p.baudRate = baudRate
	p.reader = bufio.NewReader(port)
	p.logger.Info("serial port opened", "device", p.device, "baud_rate", baudRate)
	return nil
}

// Close closes the serial port.
func (p *Port) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.port != nil {
		err := p.port.Close()
		p.port = nil
		p.reader = nil
		return err
	}
	return nil
}

// SetBaudRate changes the baud rate. Tries SetMode first, falls back to close/reopen.
func (p *Port) SetBaudRate(baudRate int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.port == nil {
		return fmt.Errorf("port not open")
	}

	mode := &goserial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   goserial.NoParity,
		StopBits: goserial.OneStopBit,
	}

	if err := p.port.SetMode(mode); err != nil {
		p.logger.Warn("SetMode failed, closing and reopening", "error", err)
		p.port.Close()
		p.port = nil

		port, err := goserial.Open(p.device, mode)
		if err != nil {
			return fmt.Errorf("reopen serial at %d: %w", baudRate, err)
		}
		p.port = port
		p.reader = bufio.NewReader(port)
	}

	p.baudRate = baudRate
	p.logger.Info("baud rate changed", "baud_rate", baudRate)
	return nil
}

// Write sends data to the serial port.
func (p *Port) Write(data string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.port == nil {
		return fmt.Errorf("port not open")
	}

	_, err := p.port.Write([]byte(data))
	return err
}

// ReadLine reads a single line (up to \n) from the serial port.
func (p *Port) ReadLine() (line string, err error) {
	p.mu.Lock()
	reader := p.reader
	p.mu.Unlock()

	if reader == nil {
		return "", fmt.Errorf("port not open")
	}

	// Recover from panics caused by reading a closed/stale bufio.Reader.
	// This can happen when the port is closed during recovery while a
	// read goroutine is still active.
	defer func() {
		if r := recover(); r != nil {
			line = ""
			err = fmt.Errorf("port read interrupted: %v", r)
		}
	}()

	line, err = reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return line, err
	}
	return line, nil
}

// BaudRate returns the current baud rate.
func (p *Port) BaudRate() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.baudRate
}

// IsOpen returns whether the port is currently open.
func (p *Port) IsOpen() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.port != nil
}
