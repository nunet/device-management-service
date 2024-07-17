package telemetry

type ObservableFactory interface {
	CreateObservable() Observable
}

type DefaultObservableFactory struct{}

func NewObservableFactory() ObservableFactory {
	return &DefaultObservableFactory{}
}

func (f *DefaultObservableFactory) CreateObservable() Observable {
	return NewObservableImpl()
}
