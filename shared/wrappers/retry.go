package wrappers

import "time"

func Retry[T any](maxRetries int, delay time.Duration, fun func() (T, error)) (T, error) {
	if maxRetries > 1 {
		result, err := fun()
		if err == nil {
			return result, nil
		}

		time.Sleep(delay)

		return Retry[T](maxRetries-1, delay, fun)
	}

	return fun()
}
