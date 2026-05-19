// Package crypto wraps filippo.io/age for ferry's needs.
//
// Invariants:
//   - identity files are written with mode 0600
//   - no function in this package returns or logs raw key bytes
package crypto

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
)

// GenerateIdentity creates a fresh X25519 identity.
func GenerateIdentity() (*age.X25519Identity, error) {
	return age.GenerateX25519Identity()
}

// WriteIdentity writes the identity to path with mode 0600.
// Parent directory is created with 0700 if missing.
func WriteIdentity(path string, id *age.X25519Identity) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.WriteString(f, id.String()+"\n"); err != nil {
		return err
	}
	return nil
}

// LoadIdentity reads an age identity from path. Accepts files containing a
// single AGE-SECRET-KEY-1... line (comments / blank lines tolerated).
func LoadIdentity(path string) (*age.X25519Identity, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		id, err := age.ParseX25519Identity(line)
		if err != nil {
			return nil, fmt.Errorf("parse key in %s: %w", path, err)
		}
		return id, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, errors.New("no age identity found in key file")
}

// EncryptStream returns a WriteCloser that encrypts to the given recipients.
// Closing the WriteCloser finalizes the age stream; callers MUST Close before
// closing the underlying writer.
func EncryptStream(w io.Writer, recipients []age.Recipient) (io.WriteCloser, error) {
	if len(recipients) == 0 {
		return nil, errors.New("no recipients")
	}
	return age.Encrypt(w, recipients...)
}

// DecryptStream returns an io.Reader that decrypts r using identities.
func DecryptStream(r io.Reader, identities []age.Identity) (io.Reader, error) {
	if len(identities) == 0 {
		return nil, errors.New("no identities")
	}
	return age.Decrypt(r, identities...)
}

// RecipientFromIdentity returns the public recipient (safe to log).
func RecipientFromIdentity(id *age.X25519Identity) age.Recipient {
	return id.Recipient()
}

// RecipientString returns the age1... public string for an identity.
func RecipientString(id *age.X25519Identity) string {
	return id.Recipient().String()
}
