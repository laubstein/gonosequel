package client

import (
	"context"
	"errors"
	"testing"

	"github.com/laubstein/gonosequel/pkg/driver"
)

func TestRunCommandUnsupported(t *testing.T) {
	c := &Client{}
	_, err := c.RunCommand(context.Background(), "db", []string{"PING"})
	if !errors.Is(err, driver.ErrUnsupported) {
		t.Errorf("expected ErrUnsupported, got %v", err)
	}
}
