package main

import (
	"context"
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	for _, mag := range magazines {
		t.Run(mag.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if _, err := NewSession(ctx, mag); err != nil {
				t.Fatal(err)
			}
		})
	}
}
