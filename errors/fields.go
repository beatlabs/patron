package errors

type WithFields interface {
	Fields() map[string]interface{}
}

type FieldsError struct {
	err    error
	fields map[string]interface{}
}

func NewFieldsError(err error, fields map[string]interface{}) error {
	return FieldsError{
		err:    err,
		fields: fields,
	}
}

func (fe FieldsError) Error() string {
	return fe.err.Error()
}

func (fe FieldsError) Fields() map[string]interface{} {
	return fe.fields
}

func (fe FieldsError) Unwrap() error {
	return fe.err
}
