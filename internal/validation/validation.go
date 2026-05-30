package validation

type Error struct {
	Message string
}

func (e Error) Error() string {
	return e.Message
}

func New(message string) Error {
	return Error{Message: message}
}
