package utils

import "errors"

func WrapErrors(err error, messages ...string) error {
	if len(messages) == 0 {
		return err
	}

	errs := make([]error, len(messages)+1)

	errs[0] = err
	for i := range messages {
		errs[i+1] = errors.New( //nolint:err113 // False positive, non static value
			messages[i],
		)
	}

	return errors.Join(errs...)
}
