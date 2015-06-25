package compose

type Comparator interface {
	Compare(expected []*Container, actual []*Container) ([]Action, error)
}

func NewComparator() Comparator {
	return &comparator{}
}

func (c *comparator) Compare(expected []*Container, actual []*Container) (res []Action, err error) {
	return
}

type comparator struct {}

