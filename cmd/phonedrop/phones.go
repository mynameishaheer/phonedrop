package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// phone: a paired phone as the UI sees it. The secret never leaves the store except to the
// request verifier (secretFor), so a phone value is safe to print or keep in UI state.
type phone struct {
	ID       string
	Name     string
	Paired   time.Time
	LastSeen time.Time
	TLS      bool // has made a signed HTTPS request; plain HTTP is refused from then on (§7)
}

type storedPhone struct {
	phone
	secret    string
	savedSeen time.Time // LastSeen as last written to disk
}

// phoneFile is the on-disk form: the secret only as base64(protect(secret)).
type phoneFile struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Secret   string    `json:"secret_protected"`
	Paired   time.Time `json:"paired"`
	LastSeen time.Time `json:"last_seen"`
	TLS      bool      `json:"tls,omitempty"`
}

// phoneStore: paired phones, persisted at path (phones.json; empty path: memory only, for
// tests). Every change is written atomically (temp file + rename) with mode 0600.
type phoneStore struct {
	path               string
	protect, unprotect func([]byte) ([]byte, error)
	now                func() time.Time

	mu     sync.Mutex
	phones map[string]*storedPhone
}

// seenEvery: LastSeen is cosmetic, so a busy phone doesn't rewrite the file on every request.
const seenEvery = time.Minute

func phonesPath() string { return filepath.Join(dataDir(), "phones.json") }

// loadPhones reads the store. Missing file: no phones. An entry that can't be read (corrupt,
// or DPAPI refuses after a Windows profile change) is skipped and logged without its secret;
// it is dropped from the file at the next save.
func loadPhones(path string, protect, unprotect func([]byte) ([]byte, error)) *phoneStore {
	s := &phoneStore{path: path, protect: protect, unprotect: unprotect, now: time.Now,
		phones: map[string]*storedPhone{}}
	if path == "" {
		return s
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("phones: %v", err)
		}
		return s
	}
	var f struct {
		Phones []json.RawMessage `json:"phones"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		log.Printf("phones: %s is not valid JSON, starting with no phones", filepath.Base(path))
		return s
	}
	for i, raw := range f.Phones {
		p, err := s.decode(raw)
		if err != nil {
			log.Printf("phones: skipping entry %d: %v", i, err)
			continue
		}
		s.phones[p.ID] = p
	}
	return s
}

func (s *phoneStore) decode(raw json.RawMessage) (*storedPhone, error) {
	var pf phoneFile
	if err := json.Unmarshal(raw, &pf); err != nil {
		return nil, errors.New("malformed")
	}
	if !validPhoneID(pf.ID) {
		return nil, errors.New("bad id")
	}
	enc, err := base64.StdEncoding.DecodeString(pf.Secret)
	if err != nil {
		return nil, errors.New("phone " + pf.ID + ": secret is not base64")
	}
	sec, err := s.unprotect(enc)
	if err != nil {
		return nil, errors.New("phone " + pf.ID + ": can't unprotect secret: " + err.Error())
	}
	if !validSecret(string(sec)) {
		return nil, errors.New("phone " + pf.ID + ": unprotected secret is malformed")
	}
	return &storedPhone{phone: phone{ID: pf.ID, Name: pf.Name, Paired: pf.Paired, LastSeen: pf.LastSeen, TLS: pf.TLS},
		secret: string(sec), savedSeen: pf.LastSeen}, nil
}

func validPhoneID(s string) bool { return len(s) == 16 && isLowerHex(s) }
func validSecret(s string) bool  { return len(s) == 64 && isLowerHex(s) }

func isLowerHex(s string) bool {
	for i := 0; i < len(s); i++ {
		if c := s[i]; !('0' <= c && c <= '9' || 'a' <= c && c <= 'f') {
			return false
		}
	}
	return true
}

// save writes the store; s.mu must be held.
func (s *phoneStore) save() error {
	if s.path == "" {
		return nil
	}
	list := make([]phoneFile, 0, len(s.phones))
	for _, p := range s.sorted() {
		enc, err := s.protect([]byte(p.secret))
		if err != nil {
			return errors.New("protecting secret: " + err.Error())
		}
		list = append(list, phoneFile{ID: p.ID, Name: p.Name, Secret: base64.StdEncoding.EncodeToString(enc),
			Paired: p.Paired, LastSeen: p.LastSeen, TLS: p.TLS})
	}
	b, err := json.MarshalIndent(struct {
		Phones []phoneFile `json:"phones"`
	}{list}, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".phones-*.tmp") // created 0600
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(f.Name(), s.path)
	}
	if err != nil {
		os.Remove(f.Name())
		return err
	}
	for _, p := range s.phones {
		p.savedSeen = p.LastSeen
	}
	return nil
}

func (s *phoneStore) sorted() []*storedPhone {
	out := make([]*storedPhone, 0, len(s.phones))
	for _, p := range s.phones {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Paired.Equal(out[j].Paired) {
			return out[i].Paired.Before(out[j].Paired)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// add stores a newly paired phone. If it can't be saved it isn't kept either, so a phone is
// never told it's paired when it won't be after a restart.
func (s *phoneStore) add(id, secret, name string) error {
	if !validPhoneID(id) || !validSecret(secret) {
		return errors.New("bad phone id or secret")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	old := s.phones[id]
	s.phones[id] = &storedPhone{phone: phone{ID: id, Name: name, Paired: now, LastSeen: now}, secret: secret}
	if err := s.save(); err != nil {
		if old != nil {
			s.phones[id] = old
		} else {
			delete(s.phones, id)
		}
		return err
	}
	return nil
}

// remove unpairs a phone; its secret stops working immediately.
func (s *phoneStore) remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.phones[id]
	if p == nil {
		return nil
	}
	delete(s.phones, id)
	if err := s.save(); err != nil {
		// Still forgotten for this run; the file catches up at the next successful save.
		log.Printf("phones: saving after remove: %v", err)
		return err
	}
	return nil
}

func (s *phoneStore) get(id string) (phone, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p := s.phones[id]; p != nil {
		return p.phone, true
	}
	return phone{}, false
}

// secretFor is for the request verifier only.
func (s *phoneStore) secretFor(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p := s.phones[id]; p != nil {
		return p.secret, true
	}
	return "", false
}

// list returns the phones, oldest pairing first, without secrets.
func (s *phoneStore) list() []phone {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []phone{}
	for _, p := range s.sorted() {
		out = append(out, p.phone)
	}
	return out
}

// touch records that a phone was just seen; written to disk at most once per seenEvery.
func (s *phoneStore) touch(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.phones[id]
	if p == nil {
		return
	}
	p.LastSeen = s.now()
	if p.LastSeen.Sub(p.savedSeen) >= seenEvery {
		if err := s.save(); err != nil {
			p.savedSeen = p.LastSeen // retry in a minute, not on every request
			log.Printf("phones: %v", err)
		}
	}
}

// setTLS marks a phone as having used HTTPS (one-way; saved immediately since it's a
// security decision).
func (s *phoneStore) setTLS(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.phones[id]
	if p == nil || p.TLS {
		return nil
	}
	p.TLS = true
	if err := s.save(); err != nil {
		log.Printf("phones: %v", err)
		return err
	}
	return nil
}
