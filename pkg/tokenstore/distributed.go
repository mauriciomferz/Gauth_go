package tokenstore

type DistributedConfig struct {
	Addresses []string
	Password  string
	DB        int
}

type DistributedStore struct{}

func NewDistributedStore(cfg DistributedConfig) *DistributedStore {
	return &DistributedStore{}
}

func (s *DistributedStore) Close() error {
	return nil
}
