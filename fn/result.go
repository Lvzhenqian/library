package fn

type Result[T any] struct {
	something T
	err       error
}

func (r *Result[T]) Unwrap() T {
	if r.err != nil {
		panic(r.err)
	}
	return r.something
}

func (r *Result[T]) Err() error {
	return r.err
}

func (r *Result[T]) IsErr() bool {
	return r.err != nil
}

func (r *Result[T]) Some() T {
	return r.something
}

func (r *Result[T]) Match(onSuccess func(value T) (T, error), onError func(err error) (T, error)) Result[T] {
	if r.IsErr() {
		return Try(onError(r.err))
	}
	return Try(onSuccess(r.something))
}

func Ok[T any](t T) Result[T] {
	return Result[T]{something: t, err: nil}
}

func Err[T any](e error) Result[T] {
	return Result[T]{err: e}
}

func Try[T any](some T, e error) Result[T] {
	if e != nil {
		return Err[T](e)
	}
	return Ok(some)
}
