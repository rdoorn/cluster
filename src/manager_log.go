package signals

import (
	"fmt"
)

func (m *Manager) log(message string, args ...interface{}) {
	/*
		pc, file, line, _ := runtime.Caller(1)
		fName := runtime.FuncForPC(pc).Name()
		//return logrus.WithField("name", name).WithField("file", path.Base(file)).WithField("line", line).WithField("func", path.Base(fName))
		message = "%s %s %s %s" + message
		//args = append(args, name)
		args = append(args, path.Base(file))
		args = append(args, line)
		args = append(args, path.Base(fName))*/

	select {
	case m.Log <- fmt.Sprintf(message, args):
	default:
	}
}

/*
func (m *Manager) handleLogs() {
	for {
		select {
		case log := <-m.logger:
			fmt.Printf(log)
		}
	}
}
*/
