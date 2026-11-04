// Package auth: password hashing (Argon2id), sessions, login limiter, admin bootstrap.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
)

// Parameters per OWASP recommendation (19 MiB, t=2, p=1): secure yet still
// compatible with the RAM target. hashSlots limits parallel hash operations
// so that concurrent logins do not blow up memory.
const (
	argonTime    = 2
	argonMemory  = 19 * 1024
	argonThreads = 1
	argonKeyLen  = 32
	saltLen      = 16
)

var hashSlots = make(chan struct{}, 2)

var ErrInvalidHash = errors.New("ungültiges Hash-Format")

func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hashSlots <- struct{}{}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	<-hashSlots
	scheduleMemoryRelease()
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrInvalidHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, ErrInvalidHash
	}
	var mem, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &t, &p); err != nil {
		return false, ErrInvalidHash
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, ErrInvalidHash
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil {
		return false, ErrInvalidHash
	}
	hashSlots <- struct{}{}
	got := argon2.IDKey([]byte(password), salt, t, mem, p, uint32(len(want)))
	<-hashSlots
	scheduleMemoryRelease()
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

var (
	dummyOnce sync.Once
	dummyHash string
)

// VerifyDummy takes the same time as a real check so that the response time
// does not reveal whether an email address exists.
func VerifyDummy(password string) {
	dummyOnce.Do(func() { dummyHash, _ = HashPassword("dummy-password-for-timing") })
	_, _ = VerifyPassword(password, dummyHash)
}

// Argon2 uses ~19 MiB per hash. Without further allocations no GC cycle runs
// afterwards, and the Go runtime keeps the memory. Shortly after the last
// hash operation it is therefore explicitly returned to the operating system
// (several logins in quick succession are coalesced).
var (
	freeMu    sync.Mutex
	freeTimer *time.Timer
)

func scheduleMemoryRelease() {
	freeMu.Lock()
	defer freeMu.Unlock()
	if freeTimer != nil {
		freeTimer.Reset(2 * time.Second)
		return
	}
	freeTimer = time.AfterFunc(2*time.Second, func() {
		debug.FreeOSMemory()
		freeMu.Lock()
		freeTimer = nil
		freeMu.Unlock()
	})
}
