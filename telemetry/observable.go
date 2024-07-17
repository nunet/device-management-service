package telemetry

type Observable interface {
	Observe(event Event)
	AddCollector(c Collector)
	GetCollectors() []Collector
}

type ObservableImpl struct {
	collectors []Collector
}

func NewObservableImpl() *ObservableImpl {
	return &ObservableImpl{}
}

func (o *ObservableImpl) Observe(event Event) {
	for _, collector := range o.collectors {
		collector.HandleEvent(event)
	}
}

func (o *ObservableImpl) AddCollector(c Collector) {
	o.collectors = append(o.collectors, c)
}

func (o *ObservableImpl) GetCollectors() []Collector {
	return o.collectors
}
