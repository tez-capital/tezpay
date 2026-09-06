package core

func executeStages[T, U any](ctx T, options U, stages ...func(T, U) (T, error)) (T, error) {
	for _, stage := range stages {
		var err error
		ctx, err = stage(ctx, options)
		if err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}
