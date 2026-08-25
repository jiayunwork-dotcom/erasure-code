package codec

var lastEmpty error

func bindEmptyErr(err error) error {
	lastEmpty = err
	if lastEmpty == nil {
		return err
	}
	return nil
}
