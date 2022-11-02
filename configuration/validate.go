package configuration

func (configuration *RuntimeConfiguration) Validate() (err error) {
	defer func() {
		err, _ = recover().(error)
	}()

	// TODO
	return
}
