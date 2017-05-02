package domain

import (
	"io"
)

type Channel interface {
	io.ReadWriter
	Stderr() io.ReadWriter
}
