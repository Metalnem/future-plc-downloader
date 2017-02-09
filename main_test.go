package main

import (
	"context"
	"testing"
	"time"
)

func TestGetProductList(t *testing.T) {
	for _, m := range magazines {
		t.Run(m.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			for {
				uid, err := m.createAnonymousUser(ctx)

				if err != nil {
					t.Fatalf("Failed to create anonymous user: %v", err)
				}

				ids, err := getProductList(ctx, uid)

				if err != nil && err.Error() == "AUT002: could not authenticate uid" {
					continue
				}

				if err != nil {
					t.Fatalf("Failed to download product list: %v", err)
				}

				if len(ids) == 0 {
					t.Fatal("Empty product list recieved")
				}

				break
			}
		})
	}
}
