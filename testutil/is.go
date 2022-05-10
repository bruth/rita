package testutil

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func NewIs(t *testing.T) *Is {
	return &Is{t}
}

type Is struct {
	t *testing.T
}

func (is *Is) Equal(a, b any) {
	if d := cmp.Diff(a, b); d != "" {
		is.t.Error(d)
	}
}

func (is *Is) Err(err error, baseErr error) {
	if err == nil {
		is.t.Error("expected error, got none")
	} else if baseErr != nil {
		if !errors.Is(err, baseErr) {
			is.t.Errorf("expected error of type %T, not %T", baseErr, err)
		}
	}
}

func (is *Is) NoErr(err error) {
	if err != nil {
		is.t.Error(err)
	}
}

func (is *Is) True(t bool) {
	if !t {
		is.t.Error("expected true")
	}
}
