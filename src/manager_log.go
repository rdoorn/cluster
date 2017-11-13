package signals

import "fmt"

func (m *Manager) log(message string, args ...interface{}) {
	select {
	case m.logger <- fmt.Sprintf(message, args):
	default:
	}
}

func (m *Manager) handleLogs() {
	for {
		select {
		case log := <-m.logger:
			fmt.Printf(log)
		}
	}
}
