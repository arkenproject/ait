package types

type StringSet interface {
	Contains(string) bool
	Add(string)
	Delete(string)
	Size() int
	ForEach(func(string) error) error
}
