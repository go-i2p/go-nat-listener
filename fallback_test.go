package nattraversal

import (
	"context"
	"net"
	"testing"
	"time"
)

// TestListenWithFallback tests the TCP listener with fallback functionality
func TestListenWithFallback(t *testing.T) {
	t.Run("Fallback when NAT traversal fails", func(t *testing.T) {
		// Note: In a real environment without NAT, this will use fallback.
		// In a test environment, we can verify the fallback behavior by
		// checking that it still creates a working listener.

		// Use a high port that's unlikely to conflict
		port := 19876

		listener, err := ListenWithFallback(port)
		if err != nil {
			t.Fatalf("ListenWithFallback failed: %v", err)
		}
		defer listener.Close()

		// Verify the listener is functional
		addr := listener.Addr()
		if addr == nil {
			t.Error("Expected non-nil address")
		}

		// Verify it implements net.Listener
		var _ net.Listener = listener

		// Verify ExternalPort returns a valid value
		extPort := listener.ExternalPort()
		if extPort <= 0 {
			t.Errorf("Expected positive external port, got %d", extPort)
		}
	})

	t.Run("Fallback listener accepts connections", func(t *testing.T) {
		port := 19877

		listener, err := ListenWithFallback(port)
		if err != nil {
			t.Fatalf("ListenWithFallback failed: %v", err)
		}
		defer listener.Close()

		// Start a goroutine to accept connections
		acceptDone := make(chan net.Conn, 1)
		go func() {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			acceptDone <- conn
		}()

		// Connect to the listener
		conn, err := net.Dial("tcp", listener.Addr().(*NATAddr).InternalAddr())
		if err != nil {
			t.Fatalf("Failed to dial listener: %v", err)
		}
		defer conn.Close()

		// Wait for accept
		select {
		case accepted := <-acceptDone:
			defer accepted.Close()
			// Connection accepted successfully
		case <-time.After(2 * time.Second):
			t.Error("Accept timed out")
		}
	})

	t.Run("IsFallback returns appropriate value", func(t *testing.T) {
		port := 19878

		listener, err := ListenWithFallback(port)
		if err != nil {
			t.Fatalf("ListenWithFallback failed: %v", err)
		}
		defer listener.Close()

		// IsFallback should return a boolean (true if NAT failed, false if succeeded)
		// We just verify it doesn't panic
		_ = listener.IsFallback()
	})

	t.Run("Close is idempotent", func(t *testing.T) {
		port := 19879

		listener, err := ListenWithFallback(port)
		if err != nil {
			t.Fatalf("ListenWithFallback failed: %v", err)
		}

		// Close multiple times should not panic
		err1 := listener.Close()
		err2 := listener.Close()

		if err1 != nil {
			t.Errorf("First close returned error: %v", err1)
		}
		if err2 != nil {
			t.Errorf("Second close returned error: %v", err2)
		}
	})
}

// TestListenWithFallbackContext tests the TCP listener with fallback and context
func TestListenWithFallbackContext(t *testing.T) {
	t.Run("Cancelled context returns error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := ListenWithFallbackContext(ctx, 19880)
		if err == nil {
			t.Error("Expected error for cancelled context")
		}
	})

	t.Run("Context with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		port := 19881
		listener, err := ListenWithFallbackContext(ctx, port)
		if err != nil {
			t.Fatalf("ListenWithFallbackContext failed: %v", err)
		}
		defer listener.Close()

		// Should work normally
		if listener.Addr() == nil {
			t.Error("Expected non-nil address")
		}
	})
}

// TestListenPacketWithFallback tests the UDP packet listener with fallback functionality
func TestListenPacketWithFallback(t *testing.T) {
	t.Run("Fallback when NAT traversal fails", func(t *testing.T) {
		port := 19882

		listener, err := ListenPacketWithFallback(port)
		if err != nil {
			t.Fatalf("ListenPacketWithFallback failed: %v", err)
		}
		defer listener.Close()

		// Verify the listener is functional
		addr := listener.Addr()
		if addr == nil {
			t.Error("Expected non-nil address")
		}

		// Verify ExternalPort returns a valid value
		extPort := listener.ExternalPort()
		if extPort <= 0 {
			t.Errorf("Expected positive external port, got %d", extPort)
		}
	})

	t.Run("Fallback packet listener sends and receives", func(t *testing.T) {
		port := 19883

		listener, err := ListenPacketWithFallback(port)
		if err != nil {
			t.Fatalf("ListenPacketWithFallback failed: %v", err)
		}
		defer listener.Close()

		// Get the packet connection
		packetConn, err := listener.Accept()
		if err != nil {
			t.Fatalf("Accept failed: %v", err)
		}

		// Create a client to send data
		clientConn, err := net.Dial("udp", listener.Addr().(*NATAddr).InternalAddr())
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer clientConn.Close()

		// Send a message
		testMsg := []byte("hello fallback")
		_, err = clientConn.Write(testMsg)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		// Receive the message
		buf := make([]byte, 1024)
		packetConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, _, err := packetConn.ReadFrom(buf)
		if err != nil {
			t.Fatalf("ReadFrom failed: %v", err)
		}

		if string(buf[:n]) != string(testMsg) {
			t.Errorf("Expected %s, got %s", testMsg, buf[:n])
		}
	})

	t.Run("IsFallback returns appropriate value", func(t *testing.T) {
		port := 19884

		listener, err := ListenPacketWithFallback(port)
		if err != nil {
			t.Fatalf("ListenPacketWithFallback failed: %v", err)
		}
		defer listener.Close()

		// IsFallback should return a boolean (true if NAT failed, false if succeeded)
		// We just verify it doesn't panic
		_ = listener.IsFallback()
	})

	t.Run("Close is idempotent", func(t *testing.T) {
		port := 19885

		listener, err := ListenPacketWithFallback(port)
		if err != nil {
			t.Fatalf("ListenPacketWithFallback failed: %v", err)
		}

		// Close multiple times should not panic
		err1 := listener.Close()
		err2 := listener.Close()

		if err1 != nil {
			t.Errorf("First close returned error: %v", err1)
		}
		if err2 != nil {
			t.Errorf("Second close returned error: %v", err2)
		}
	})
}

// TestListenPacketWithFallbackContext tests the UDP packet listener with fallback and context
func TestListenPacketWithFallbackContext(t *testing.T) {
	t.Run("Cancelled context returns error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := ListenPacketWithFallbackContext(ctx, 19886)
		if err == nil {
			t.Error("Expected error for cancelled context")
		}
	})

	t.Run("Context with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		port := 19887
		listener, err := ListenPacketWithFallbackContext(ctx, port)
		if err != nil {
			t.Fatalf("ListenPacketWithFallbackContext failed: %v", err)
		}
		defer listener.Close()

		// Should work normally
		if listener.Addr() == nil {
			t.Error("Expected non-nil address")
		}
	})
}

// TestFallbackModeProperties tests specific properties of fallback mode
func TestFallbackModeProperties(t *testing.T) {
	t.Run("Fallback listener NATAddr has same internal and external", func(t *testing.T) {
		port := 19888

		listener, err := ListenWithFallback(port)
		if err != nil {
			t.Fatalf("ListenWithFallback failed: %v", err)
		}
		defer listener.Close()

		if listener.IsFallback() {
			addr := listener.Addr().(*NATAddr)
			if addr.InternalAddr() != addr.ExternalAddr() {
				t.Errorf("In fallback mode, internal (%s) and external (%s) should be the same",
					addr.InternalAddr(), addr.ExternalAddr())
			}
		}
	})

	t.Run("Fallback packet listener NATAddr has same internal and external", func(t *testing.T) {
		port := 19889

		listener, err := ListenPacketWithFallback(port)
		if err != nil {
			t.Fatalf("ListenPacketWithFallback failed: %v", err)
		}
		defer listener.Close()

		if listener.IsFallback() {
			addr := listener.Addr().(*NATAddr)
			if addr.InternalAddr() != addr.ExternalAddr() {
				t.Errorf("In fallback mode, internal (%s) and external (%s) should be the same",
					addr.InternalAddr(), addr.ExternalAddr())
			}
		}
	})
}
