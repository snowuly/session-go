# Session Go

```go
func Register(string, Provider)

type Provider interface {
	SessionInit(sid string) (Session, error)
	SessionRead(sid string) (Session, error)
	SessionDestroy(sid string)
	GC(maxLifetime int)
}


type Session interface {
	Get(string) interface{}
	Set(string, interface{})
	Del(string)
	SessionId() string
}
```
