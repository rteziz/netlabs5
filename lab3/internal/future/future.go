package future

import (
	"context"
	"sync"
)

// Future хранит результат операции и функции, ожидающие её завершения.
type Future[T any] struct {
	mu        sync.Mutex
	done      bool
	value     T
	err       error
	callbacks []func(T, error)
}

func New[T any]() *Future[T] { return &Future[T]{} }

func (f *Future[T]) Resolve(value T, err error) {
	f.mu.Lock()
	if f.done {
		f.mu.Unlock()
		return
	}
	f.done, f.value, f.err = true, value, err
	// Вызываем подписчиков после снятия блокировки: они могут подписаться снова.
	callbacks := f.callbacks
	f.callbacks = nil
	f.mu.Unlock()
	for _, callback := range callbacks {
		callback(value, err)
	}
}

func (f *Future[T]) subscribe(callback func(T, error)) {
	f.mu.Lock()
	if !f.done {
		f.callbacks = append(f.callbacks, callback)
		f.mu.Unlock()
		return
	}
	value, err := f.value, f.err
	f.mu.Unlock()
	callback(value, err)
}

func Ready[T any](value T, err error) *Future[T] {
	f := New[T]()
	f.Resolve(value, err)
	return f
}

func Then[A, B any](source *Future[A], next func(A) *Future[B]) *Future[B] {
	result := New[B]()
	source.subscribe(func(value A, err error) {
		if err != nil {
			var zero B
			result.Resolve(zero, err)
			return
		}
		// Результат следующего шага завершит всю цепочку.
		next(value).subscribe(result.Resolve)
	})
	return result
}

func Map[A, B any](source *Future[A], transform func(A) (B, error)) *Future[B] {
	return Then(source, func(value A) *Future[B] {
		mapped, err := transform(value)
		return Ready(mapped, err)
	})
}

func All[T any](inputs ...*Future[T]) *Future[[]T] {
	result := New[[]T]()
	values := make([]T, len(inputs))
	if len(inputs) == 0 {
		result.Resolve(values, nil)
		return result
	}
	var mu sync.Mutex
	remaining := len(inputs)
	for i, input := range inputs {
		index := i
		input.subscribe(func(value T, err error) {
			if err != nil {
				result.Resolve(nil, err)
				return
			}
			mu.Lock()
			// Порядок результатов определяется порядком входов, даже если ответы пришли иначе.
			values[index] = value
			remaining--
			done := remaining == 0
			mu.Unlock()
			if done {
				result.Resolve(values, nil)
			}
		})
	}
	return result
}

func (f *Future[T]) Await(ctx context.Context) (T, error) {
	type outcome struct {
		value T
		err   error
	}
	// Буфер принимает ответ и после выхода из Await по отмене контекста.
	ready := make(chan outcome, 1)
	f.subscribe(func(value T, err error) { ready <- outcome{value, err} })
	select {
	case result := <-ready:
		return result.value, result.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

func (f *Future[T]) OnComplete(callback func(T, error)) {
	f.subscribe(callback)
}
