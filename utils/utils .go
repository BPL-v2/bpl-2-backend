package utils

func Map[A any, B any](input []A, mapper func(A) B) []B {
	output := make([]B, len(input))
	for i, item := range input {
		output[i] = mapper(item)
	}
	return output
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
