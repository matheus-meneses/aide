package notification

type Notifier interface {
	Notify(title, body string) error
}

type NoopNotifier struct{}

func (n *NoopNotifier) Notify(_, _ string) error {
	return nil
}

// MultiNotifier fans a notification out to several notifiers.
type MultiNotifier struct {
	notifiers []Notifier
}

func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{notifiers: notifiers}
}

func (m *MultiNotifier) Notify(title, body string) error {
	var firstErr error
	for _, n := range m.notifiers {
		if err := n.Notify(title, body); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
