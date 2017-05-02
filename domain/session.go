package domain

type Session interface {
	Shell() error
	Exec(cmd string) (int, error)
	SendRequest(name string, wantReply bool, payload []byte) (bool, error)
}
