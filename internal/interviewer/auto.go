package interviewer

// AutoApproveInterviewer always selects YES or the first option.
type AutoApproveInterviewer struct{}

func NewAutoApproveInterviewer() *AutoApproveInterviewer {
	return &AutoApproveInterviewer{}
}

func (a *AutoApproveInterviewer) Ask(q Question) Answer {
	switch q.Type {
	case YesNo, Confirmation:
		return Answer{Value: AnswerYes}
	case MultipleChoice:
		if len(q.Options) > 0 {
			o := q.Options[0]
			return Answer{Value: AnswerValue(o.Key), SelectedOption: &o}
		}
	}
	return Answer{Value: AnswerValue("auto-approved"), Text: "auto-approved"}
}

func (a *AutoApproveInterviewer) Inform(message, stage string) {}

// QueueInterviewer reads from a pre-filled answer queue.
type QueueInterviewer struct {
	answers []Answer
	pos     int
}

func NewQueueInterviewer(answers ...Answer) *QueueInterviewer {
	return &QueueInterviewer{answers: answers}
}

func (q *QueueInterviewer) Ask(question Question) Answer {
	if q.pos < len(q.answers) {
		a := q.answers[q.pos]
		q.pos++
		return a
	}
	return Answer{Value: AnswerSkipped}
}

func (q *QueueInterviewer) Inform(message, stage string) {}
