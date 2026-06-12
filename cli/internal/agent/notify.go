package agent

type Notifier interface {
	Notify(title, body string) error
}

type NoopNotifier struct{}

func (n *NoopNotifier) Notify(_, _ string) error {
	return nil
}
