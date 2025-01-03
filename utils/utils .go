package utils

func Map[A any, B any](input []A, mapper func(A) B) []B {
	output := make([]B, len(input))
	for i, item := range input {
		output[i] = mapper(item)
	}
	return output
}

func Filter[A any](input []A, filter func(A) bool) []A {
	output := make([]A, 0, len(input))
	for _, item := range input {
		if filter(item) {
			output = append(output, item)
		}
	}
	return output
}

func Contains[A comparable](input []A, item A) bool {
	for _, i := range input {
		if i == item {
			return true
		}
	}
	return false
}

func Keys[A comparable, B any](input map[A]B) []A {
	keys := make([]A, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	return keys
}

func Values[A comparable, B any](input map[A]B) []B {
	values := make([]B, 0, len(input))
	for _, value := range input {
		values = append(values, value)
	}
	return values
}

func Uniques[A comparable](input []A) []A {
	ids := make(map[A]bool)
	for _, item := range input {
		ids[item] = true
	}
	uniques := make([]A, 0, len(ids))
	for id := range ids {
		uniques = append(uniques, id)
	}
	return uniques
}

func BatchIterator[A any](input []A, batchSize int) <-chan []A {
	ch := make(chan []A)
	go func() {
		defer close(ch)
		for i := 0; i < len(input); i += batchSize {
			end := i + batchSize
			if end > len(input) {
				end = len(input)
			}
			ch <- input[i:end]
		}
	}()
	return ch
}
