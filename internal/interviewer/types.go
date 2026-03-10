package interviewer

// QuestionType determines the UI and valid answers.
type QuestionType int

const (
	YesNo          QuestionType = iota
	MultipleChoice
	Freeform
	Confirmation
)

// Option represents a choice in a multiple-choice question.
type Option struct {
	Key   string
	Label string
}

// Question is presented to a human via the Interviewer.
type Question struct {
	Text    string
	Type    QuestionType
	Options []Option
	Stage   string
}

// AnswerValue represents special answer states.
type AnswerValue string

const (
	AnswerYes     AnswerValue = "yes"
	AnswerNo      AnswerValue = "no"
	AnswerSkipped AnswerValue = "skipped"
	AnswerTimeout AnswerValue = "timeout"
)

// Answer is the human's response to a question.
type Answer struct {
	Value          AnswerValue
	SelectedOption *Option
	Text           string
}

// Interviewer is the interface for human interaction.
type Interviewer interface {
	Ask(q Question) Answer
	Inform(message, stage string)
}
