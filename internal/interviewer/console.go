package interviewer

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ConsoleInterviewer reads from stdin.
type ConsoleInterviewer struct {
	reader *bufio.Reader
}

func NewConsoleInterviewer() *ConsoleInterviewer {
	return &ConsoleInterviewer{reader: bufio.NewReader(os.Stdin)}
}

func (c *ConsoleInterviewer) Ask(q Question) Answer {
	fmt.Printf("\n[?] %s\n", q.Text)

	switch q.Type {
	case MultipleChoice:
		for _, opt := range q.Options {
			fmt.Printf("  [%s] %s\n", opt.Key, opt.Label)
		}
		fmt.Print("Select: ")
		line := c.readLine()
		line = strings.TrimSpace(line)

		for _, opt := range q.Options {
			if strings.EqualFold(line, opt.Key) || strings.EqualFold(line, opt.Label) {
				o := opt
				return Answer{Value: AnswerValue(opt.Key), SelectedOption: &o}
			}
		}
		if len(q.Options) > 0 {
			o := q.Options[0]
			return Answer{Value: AnswerValue(o.Key), SelectedOption: &o}
		}
		return Answer{Value: AnswerSkipped}

	case YesNo, Confirmation:
		fmt.Print("[Y/N]: ")
		line := strings.ToLower(strings.TrimSpace(c.readLine()))
		if line == "y" || line == "yes" {
			return Answer{Value: AnswerYes}
		}
		return Answer{Value: AnswerNo}

	case Freeform:
		fmt.Print("> ")
		line := c.readLine()
		return Answer{Text: strings.TrimSpace(line), Value: AnswerValue(strings.TrimSpace(line))}

	default:
		return Answer{Value: AnswerSkipped}
	}
}

func (c *ConsoleInterviewer) Inform(message, stage string) {
	fmt.Printf("[i] [%s] %s\n", stage, message)
}

func (c *ConsoleInterviewer) readLine() string {
	line, _ := c.reader.ReadString('\n')
	return strings.TrimRight(line, "\n\r")
}
