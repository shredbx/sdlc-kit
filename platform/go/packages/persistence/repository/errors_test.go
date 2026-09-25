package repository

import (
	"errors"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	t.Run("ErrNotFound_is_error", func(t *testing.T) {
		var err error = ErrNotFound
		if err.Error() != "not found" {
			t.Errorf("ErrNotFound.Error() = %q, want %q", err.Error(), "not found")
		}
	})

	t.Run("ErrConflict_is_error", func(t *testing.T) {
		var err error = ErrConflict
		if err.Error() != "conflict" {
			t.Errorf("ErrConflict.Error() = %q, want %q", err.Error(), "conflict")
		}
	})

	t.Run("ErrValidation_is_error", func(t *testing.T) {
		var err error = ErrValidation
		if err.Error() != "validation failed" {
			t.Errorf("ErrValidation.Error() = %q, want %q", err.Error(), "validation failed")
		}
	})

	t.Run("sentinels_are_distinct", func(t *testing.T) {
		if errors.Is(ErrNotFound, ErrConflict) {
			t.Error("ErrNotFound should not match ErrConflict")
		}
		if errors.Is(ErrNotFound, ErrValidation) {
			t.Error("ErrNotFound should not match ErrValidation")
		}
		if errors.Is(ErrConflict, ErrValidation) {
			t.Error("ErrConflict should not match ErrValidation")
		}
	})
}

func TestNewNotFoundError(t *testing.T) {
	t.Run("creates_typed_error", func(t *testing.T) {
		err := NewNotFoundError("property", "abc-123")
		if err == nil {
			t.Fatal("NewNotFoundError returned nil")
		}
	})

	t.Run("formats_entity_and_id", func(t *testing.T) {
		err := NewNotFoundError("property", "abc-123")
		want := `property with id "abc-123" not found`
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("unwraps_to_ErrNotFound", func(t *testing.T) {
		err := NewNotFoundError("booking", "xyz")
		if !errors.Is(err, ErrNotFound) {
			t.Error("NewNotFoundError should unwrap to ErrNotFound")
		}
	})

	t.Run("does_not_match_ErrConflict", func(t *testing.T) {
		err := NewNotFoundError("booking", "xyz")
		if errors.Is(err, ErrConflict) {
			t.Error("NewNotFoundError should not match ErrConflict")
		}
	})

	t.Run("can_type_assert", func(t *testing.T) {
		err := NewNotFoundError("user", "42")
		var nfe *NotFoundError
		if !errors.As(err, &nfe) {
			t.Fatal("errors.As should succeed for *NotFoundError")
		}
		if nfe.Entity != "user" {
			t.Errorf("Entity = %q, want %q", nfe.Entity, "user")
		}
		if nfe.ID != "42" {
			t.Errorf("ID = %q, want %q", nfe.ID, "42")
		}
	})

	t.Run("empty_entity_and_id", func(t *testing.T) {
		err := NewNotFoundError("", "")
		want := ` with id "" not found`
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
		if !errors.Is(err, ErrNotFound) {
			t.Error("should still unwrap to ErrNotFound")
		}
	})
}
