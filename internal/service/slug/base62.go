package slug

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Generator interface {
	Next() (string, error)
}

type Base62Generator struct {
	db           *pgxpool.Pool
	sequenceName string
}

func NewBase62(db *pgxpool.Pool, sequenceName string) *Base62Generator {
	if sequenceName == "" {
		sequenceName = "slug_seq"
	}
	return &Base62Generator{db: db, sequenceName: sequenceName}
}

func (g *Base62Generator) Next() (string, error) {
	if g.db == nil {
		return "", fmt.Errorf("db is nil")
	}
	var id int64
	// Note: sequence name cannot be parameterized in nextval(), so we build query string.
	q := fmt.Sprintf("SELECT nextval('%s')", g.sequenceName)
	if err := g.db.QueryRow(context.Background(), q).Scan(&id); err != nil {
		return "", err
	}
	enc := encodeBase62(uint64(id))
	// ensure minimum length of 4 by left-padding with '0'
	for len(enc) < 4 {
		enc = string(base62Alphabet[0]) + enc
	}
	return enc, nil
}

const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func encodeBase62(n uint64) string {
	if n == 0 {
		return string(base62Alphabet[0])
	}
	buf := make([]byte, 0, 11) // enough for 64-bit base62
	for n > 0 {
		r := n % 62
		buf = append(buf, base62Alphabet[r])
		n /= 62
	}
	// reverse
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
