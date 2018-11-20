package session

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"sync"
	"time"
)

var providers = make(map[string]Provider)

type Provider interface {
	SessionInit(sid string) (Session, error)
	SessionRead(sid string) (Session, error)
	SessionDestroy(sid string)
	GC(maxLifetime int)
}

func Register(name string, provider Provider) {
	if provider == nil {
		panic("nil provider")
	}
	if _, dup := providers[name]; dup {
		panic("dup provider name")
	}
	providers[name] = provider
}

type Session interface {
	Get(string) interface{}
	Set(string, interface{})
	Del(string)
	SessionId() string
}

type Manager struct {
	cookieName  string
	provider    Provider
	maxLifetime int
	lock        sync.Mutex
}

func NewManager(name, cookieName string, maxLifetime int) *Manager {
	provider, ok := providers[name]
	if !ok {
		panic("provider not exists")
	}
	return &Manager{cookieName: cookieName, provider: provider, maxLifetime: maxLifetime}
}

func (manager *Manager) GC() {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	manager.provider.GC(manager.maxLifetime)
	time.AfterFunc(time.Duration(manager.maxLifetime)*time.Second, func() { manager.GC() })
}

func (manager *Manager) sessionId() string {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func (manager *Manager) SessionStart(w http.ResponseWriter, r *http.Request) (session Session) {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	cookie, err := r.Cookie(manager.cookieName)
	if err != nil || cookie.Value == "" {
		sid := manager.sessionId()
		session, _ = manager.provider.SessionInit(sid)
		cookie := &http.Cookie{Name: manager.cookieName, Value: sid, Path: "/", MaxAge: manager.maxLifetime, HttpOnly: true, SameSite: http.SameSiteLaxMode}
		http.SetCookie(w, cookie)
	} else {
		session, _ = manager.provider.SessionRead(cookie.Value)
	}
	return
}
func (manager *Manager) SessionDestroy(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(manager.cookieName)
	if err != nil || cookie.Value == "" {
		return
	}
	manager.lock.Lock()
	defer manager.lock.Unlock()
	manager.provider.SessionDestroy(cookie.Value)
	cookie = &http.Cookie{Name: manager.cookieName, Path: "/", MaxAge: -1, HttpOnly: true}
	http.SetCookie(w, cookie)
}
