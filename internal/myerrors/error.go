package myerrors

type RequestError struct {
	Message string
}

func (r *RequestError) Error() string {
	return r.Message
}